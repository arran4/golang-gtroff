package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/arran4/golang-gtroff/internal/atomicfile"
	"github.com/arran4/golang-gtroff/internal/sourceio"
	"github.com/arran4/golang-gtroff/scriptapi"
	"github.com/arran4/golang-gtroff/source"
	"io"
	"reflect"
	"strings"
	"time"
)

const replHelp = `Tengo REPL: native cells retain variables/functions and the working document.
Multiline expressions finish when native syntax is complete. Use doc.emit or
emit_json for stdout; expressions are not implicitly printed. Prompts use stderr.
:help       Show this help
:cancel     Discard incomplete multiline input
:pending    Inspect the pending document byte diff after a publication failure
:retry      Retry the same pending output/write batch (stream prefixes may repeat)
:discard    Accept pending memory state, discarding its output/write requests
:commit     Explicitly commit current original-file changes; requires --in-place
:quit       End the session without additional writes
Failed cells roll back globals/document/output; successful cells publish separately.
A publication failure retains its pending state and blocks further cells until
retry/discard. Original guards persist; only successful commits advance baselines.
`

type replPublicationError struct{ code int }

func (e replPublicationError) Error() string { return "pending REPL publication failed" }

func runScriptREPL(c scriptCLI, files []string, e Environment) int {
	if len(files) > 1 || len(files) == 1 && (files[0] == "" || files[0] == "-") {
		return usage(e, "REPL expects at most one named DOCUMENT; stdin belongs to cells")
	}
	if code := validateScriptConfig(c, e); code != ExitOK {
		return code
	}
	if c.maxCells < 1 || c.maxCells > 10000 || c.maxSessionBytes < 1 || c.maxSessionBytes > 64<<20 {
		return usage(e, "REPL session limits outside supported ranges")
	}
	if e.ScriptSession == nil {
		return failure(e, "Tengo REPL is not included in this build; build: go -C script build -o ../gtroff ./cli")
	}
	if sourceio.IsNil(e.In) {
		return failure(e, "nil REPL input")
	}
	ctx := e.Context
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return failure(e, err.Error())
	}
	var set *source.Set
	var err error
	inputFile := ""
	paths := map[string]string{}
	if len(files) == 1 {
		inputFile = files[0]
		set, err = loadSources(inputFile, c.input, e)
		if err != nil {
			return editFailure(e, err)
		}
		paths = editSourcePaths(inputFile, c.input, e, set)
	}
	c.options.AllowWriteRequests = c.writeTarget != ""
	c.options.Edit.Source = c.input.Options
	c.options.Edit.Strict = c.input.Strict
	c.options.Edit.MaxBytes = c.input.MaxBytes
	c.options.MaxAllocs = int64(c.maxAllocs)
	c.options.Duration = time.Duration(c.durationMS) * time.Millisecond
	session, err := e.ScriptSession(ctx, scriptapi.Request{Sources: set}, scriptapi.SessionOptions{Cell: c.options, MaxCells: c.maxCells, MaxSessionBytes: c.maxSessionBytes})
	if err != nil {
		return scriptFailure(e, err)
	}
	if nilScriptSession(session) {
		return failure(e, "nil script session")
	}
	guards := scriptSourceGuards(set, paths)
	lastPublish := c
	lastPublish.inPlace = false
	publish := func() int {
		err := session.Publish(ctx, func(ctx context.Context, result *scriptapi.Result) error {
			code := publishScriptWithGuards(lastPublish, paths, guards, result, e)
			if code != ExitOK {
				return replPublicationError{code}
			}
			// Only a successful explicit original-file commit acknowledges new guards.
			if lastPublish.inPlace && !lastPublish.dryRun && result.Working != nil {
				var next []atomicfile.SourceGuard
				for _, g := range result.Working.Guards() {
					if path := paths[g.Name]; path != "" {
						next = append(next, atomicfile.SourceGuard{Path: path, Digest: g.Digest, Bytes: g.Bytes})
					}
				}
				guards = next
			}
			return nil
		})
		if err == nil {
			return ExitOK
		}
		var failed replPublicationError
		if errors.As(err, &failed) {
			return failed.code
		}
		return scriptFailure(e, err)
	}
	reader := bufio.NewReaderSize(scriptContextReader{ctx, sourceio.NewReader(e.In, func(n, size int, cause error) error {
		return errors.Join(fmt.Errorf("invalid REPL reader count %d/%d", n, size), cause)
	})}, 4096)
	var cell []byte
	consumed, attempts, status := 0, 0, ExitOK
	for {
		if err := ctx.Err(); err != nil {
			return failure(e, err.Error())
		}
		if c.prompt {
			prompt := "gtroff> "
			if len(cell) > 0 {
				prompt = "...> "
			}
			if _, err := io.WriteString(e.Err, prompt); err != nil {
				return failure(e, err.Error())
			}
		}
		line, readErr := readREPLLine(reader, min(c.options.MaxScriptBytes-len(cell), c.maxSessionBytes-consumed))
		consumed += len(line)
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return failure(e, readErr.Error())
		}
		if len(line) == 0 && errors.Is(readErr, io.EOF) {
			if len(cell) > 0 {
				return max(status, invalid(e, "incomplete Tengo cell at EOF"))
			}
			return status
		}
		command := strings.TrimSpace(string(line))
		control := len(cell) == 0 && strings.HasPrefix(command, ":") || command == ":cancel"
		if control {
			switch command {
			case ":quit":
				return status
			case ":help":
				if err := writeEditPreview(e.Err, []byte(replHelp)); err != nil {
					return failure(e, err.Error())
				}
				continue
			case ":cancel":
				cell = nil
				continue
			case ":retry":
				if len(cell) > 0 {
					status = max(status, invalid(e, "cancel incomplete input before retry"))
					continue
				}
				status = max(status, publish())
				continue
			case ":discard":
				if err := session.Publish(ctx, func(context.Context, *scriptapi.Result) error { return nil }); err != nil {
					status = max(status, scriptFailure(e, err))
				}
				continue
			case ":pending":
				pending, err := session.Pending(ctx)
				if err != nil {
					return scriptFailure(e, err)
				}
				if pending == nil {
					status = max(status, invalid(e, "no pending successful cell"))
					continue
				}
				if pending.Working != nil {
					if err := pending.Working.WritePreview(ctx, e.Out, "diff", c.options.MaxOutputBytes); err != nil {
						return failure(e, err.Error())
					}
				}
				continue
			case ":commit":
				if !c.inPlace {
					status = max(status, invalid(e, "commit requires explicit --in-place authorization"))
					continue
				}
				if len(cell) > 0 {
					status = max(status, invalid(e, "cancel incomplete input before commit"))
					continue
				}
				line = []byte("\n") // Empty native cell captures current state without executing host code.
			default:
				status = max(status, invalid(e, "unknown REPL control; use :help"))
				continue
			}
		}
		cell = append(cell, line...)
		ready, err := session.Ready(ctx, scriptapi.Cell{Code: cell})
		if err != nil {
			status = max(status, scriptFailure(e, err))
			cell = nil
			if errors.Is(err, scriptapi.ErrLimit) {
				return status
			}
			continue
		}
		if !ready {
			if errors.Is(readErr, io.EOF) {
				return max(status, invalid(e, "incomplete Tengo cell at EOF"))
			}
			continue
		}
		if strings.TrimSpace(string(cell)) == "" && command != ":commit" {
			cell = nil
			continue
		}
		attempts++
		if attempts > c.maxCells {
			return max(status, failure(e, "REPL cell limit exceeded"))
		}
		_, err = session.Eval(ctx, scriptapi.Cell{Name: fmt.Sprintf("<repl-%d>", attempts), Code: cell})
		cell = nil
		if err != nil {
			status = max(status, scriptFailure(e, err))
			continue
		}
		lastPublish = c
		lastPublish.inPlace = command == ":commit"
		status = max(status, publish())
		if errors.Is(readErr, io.EOF) {
			return status
		}
	}
}
func readREPLLine(reader *bufio.Reader, limit int) ([]byte, error) {
	var line []byte
	for {
		part, err := reader.ReadSlice('\n')
		if len(part) > limit-len(line) {
			return nil, scriptapi.ErrLimit
		}
		line = append(line, part...)
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		return line, err
	}
}

func nilScriptSession(s scriptapi.Session) bool {
	if s == nil {
		return true
	}
	v := reflect.ValueOf(s)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	}
	return false
}
