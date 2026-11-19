// Package script embeds Tengo in an optional module over captured roff documents.
// It has no ambient filesystem, network, shell or import capabilities. The
// interpreter runs in the host process; its allocation/deadline limits are not
// an OS sandbox or a hard heap-byte quota.
package script

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/arran4/golang-gtroff/edit"
	"github.com/arran4/golang-gtroff/extract"
	"github.com/arran4/golang-gtroff/query"
	"github.com/arran4/golang-gtroff/scriptapi"
	"github.com/arran4/golang-gtroff/source"
	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/parser"
	"github.com/d5/tengo/v2/token"
)

// Request, Options and results share an engine-free port with the optional CLI.
type Request = scriptapi.Request
type Options = scriptapi.Options
type Result = scriptapi.Result
type Stats = scriptapi.Stats
type WriteRequest = scriptapi.WriteRequest

var (
	ErrOptions  = scriptapi.ErrOptions
	ErrLimit    = scriptapi.ErrLimit
	ErrDenied   = scriptapi.ErrDenied
	ErrDocument = scriptapi.ErrDocument
)

type runtime struct {
	ctx          context.Context
	o            Options
	working      *edit.Working
	output       []byte
	writes       []WriteRequest
	writeVersion uint64
	writeWorking *edit.Working
	calls        int
	steps        int
	hostError    error
	diagnostics  []edit.Diagnostic
	created      bool
}

func defaults(o Options) (Options, error) {
	if o.MaxScriptBytes <= 0 {
		o.MaxScriptBytes = 64 << 10
	}
	if o.MaxTokens <= 0 {
		o.MaxTokens = 20000
	}
	if o.MaxNesting <= 0 {
		o.MaxNesting = 128
	}
	if o.MaxConstants <= 0 {
		o.MaxConstants = 4096
	}
	if o.MaxHostCalls <= 0 {
		o.MaxHostCalls = 1000
	}
	if o.MaxOutputBytes <= 0 {
		o.MaxOutputBytes = 16 << 20
	}
	if o.MaxSteps <= 0 {
		o.MaxSteps = 100000
	}
	if o.MaxAllocs <= 0 {
		o.MaxAllocs = 100000
	}
	if o.Duration <= 0 {
		o.Duration = time.Second
	}
	if o.MaxScriptBytes > 1<<20 || o.MaxTokens > 100000 || o.MaxNesting > 256 || o.MaxConstants > 100000 || o.MaxHostCalls > 100000 || o.MaxOutputBytes > 64<<20 || o.MaxAllocs > 1000000 || o.MaxSteps > 10000000 || o.Duration > 30*time.Second {
		return o, ErrOptions
	}
	return o, nil
}

// Run compiles the native Tengo grammar and executes with explicit host objects.
// Each invocation starts an independent working snapshot. It does not change the
// supplied source.Set. Native map iteration order is unspecified; host query
// arrays and emitted JSON field ordering are deterministic. No modules or file
// imports are installed. All output and write intents stay staged until success.
func Run(ctx context.Context, request Request, o Options) (result *Result, err error) {
	result = &Result{}
	defer func() {
		if err != nil && !errors.Is(err, ErrLimit) && (errors.Is(err, tengo.ErrObjectAllocLimit) || errors.Is(err, tengo.ErrStackOverflow) || errors.Is(err, tengo.ErrStringLimit) || errors.Is(err, tengo.ErrBytesLimit) || errors.Is(err, edit.ErrLimit) || errors.Is(err, extract.ErrLimit) || errors.Is(err, query.ErrLimit) || errors.Is(err, source.ErrLimit)) {
			err = errors.Join(ErrLimit, err)
		}
	}()
	if ctx == nil {
		ctx = context.Background()
	}
	if err = ctx.Err(); err != nil {
		return result, err
	}
	if request.InputError != nil {
		return result, request.InputError
	}
	o, err = defaults(o)
	if err != nil {
		return result, err
	}
	if len(request.Code) > o.MaxScriptBytes {
		return result, ErrLimit
	}
	if request.Name == "" {
		request.Name = "<script>"
	}
	ctx, cancel := context.WithTimeout(ctx, o.Duration)
	defer cancel()
	r := &runtime{ctx: ctx, o: o}
	if request.Sources != nil {
		r.working, err = edit.New(request.Sources, o.Edit)
		if err != nil {
			return result, err
		}
	}
	started := time.Now()
	files := parser.NewFileSet()
	file := files.AddFile(request.Name, -1, len(request.Code))
	if err = preflightScript(ctx, file, request.Code, o); err != nil {
		return result, err
	}
	parsed, err := parser.NewParser(file, request.Code, nil).ParseFile()
	if err != nil {
		return result, err
	}
	if err = nativeShape(ctx, parsed, 4*o.MaxTokens); err != nil {
		return result, err
	}
	symbols := tengo.NewSymbolTable()
	compiler := tengo.NewCompiler(file, symbols, nil, tengo.NewModuleMap(), nil)
	compiler.EnableFileImport(false)
	// Define after NewCompiler installs built-ins, keeping the host slot stable.
	symbol := symbols.Define("doc")
	globals := make([]tengo.Object, tengo.GlobalsSize)
	globals[symbol.Index] = r.documentObject()
	checkpoint := symbols.Define(checkpointName)
	globals[checkpoint.Index] = &tengo.UserFunction{Name: checkpointName, Value: r.checkpoint}
	if err = instrument(ctx, parsed, 2*o.MaxTokens); err != nil {
		return result, err
	}
	if err = compiler.Compile(parsed); err != nil {
		return result, err
	}
	bytecode := compiler.Bytecode()
	if err = nativeConstantBounds(bytecode); err != nil {
		return result, err
	}
	bytecode.RemoveDuplicates()
	result.Stats.Constants = bytecode.CountObjects()
	if result.Stats.Constants > o.MaxConstants {
		return result, ErrLimit
	}
	result.Stats.CompileDuration = time.Since(started)
	if err = ctx.Err(); err != nil {
		return result, err
	}
	vm := tengo.NewVM(bytecode, globals, o.MaxAllocs)
	started = time.Now()
	err = runVM(ctx, vm)
	result.Stats.RunDuration = time.Since(started)
	result.Stats.HostCalls = r.calls
	result.Stats.Steps = r.steps
	result.Stats.OutputBytes = len(r.output)
	if r.hostError != nil && !errors.Is(err, r.hostError) {
		err = errors.Join(err, r.hostError)
	}
	if err != nil {
		return result, err
	}
	if len(r.writes) > 0 && (r.working != r.writeWorking || r.working.Version() != r.writeVersion) {
		return result, edit.ErrStale
	}
	result.Diagnostics = append([]edit.Diagnostic(nil), r.diagnostics...)
	if r.working != nil {
		result.SourceDiagnostics = r.working.Sources().Diagnostics()
	}
	result.Created = r.created
	result.Working = r.working
	result.Output = append([]byte(nil), r.output...)
	for _, w := range r.writes {
		w.Data = append([]byte(nil), w.Data...)
		result.Writes = append(result.Writes, w)
	}
	return result, nil
}

func runVM(ctx context.Context, vm *tengo.VM) error {
	done := make(chan error, 1)
	go func() {
		defer func() {
			if p := recover(); p != nil {
				done <- fmt.Errorf("Tengo execution panic: %v", p)
			}
		}()
		done <- vm.Run()
	}()
	select {
	case err := <-done:
		if cancelled := ctx.Err(); cancelled != nil {
			return cancelled
		}
		return err
	case <-ctx.Done():
		vm.Abort()
		<-done
		return ctx.Err()
	}
}

func preflightScript(ctx context.Context, file *parser.SourceFile, code []byte, o Options) error {
	scanner := parser.NewScanner(file, code, nil, 0)
	nesting := 0
	for count := 0; ; count++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if count >= o.MaxTokens {
			return ErrLimit
		}
		tok, literal, pos := scanner.Scan()
		if tok == token.Ident && reservedIdentifier(literal) {
			return fmt.Errorf("%w: reserved host identifier at %s", ErrDenied, file.Position(pos))
		}
		if tok == token.EOF {
			return nil
		}
		switch tok {
		case token.LParen, token.LBrack, token.LBrace:
			nesting++
		case token.RParen, token.RBrack, token.RBrace:
			nesting--
		}
		if nesting > o.MaxNesting {
			return fmt.Errorf("%w: script nesting at %s", ErrLimit, file.Position(pos))
		}
	}
}

// function wraps host failures so a script cannot hide a denied capability or
// resource failure by converting it to an ordinary Tengo value.
func (r *runtime) function(name string, call tengo.CallableFunc) *tengo.UserFunction {
	return &tengo.UserFunction{Name: name, Value: func(args ...tengo.Object) (tengo.Object, error) {
		if err := r.ctx.Err(); err != nil {
			r.hostError = err
			return nil, err
		}
		r.calls++
		if r.calls > r.o.MaxHostCalls {
			r.hostError = ErrLimit
			return nil, ErrLimit
		}
		value, err := call(args...)
		if err != nil && r.hostError == nil {
			r.hostError = err
		}
		return value, err
	}}
}
