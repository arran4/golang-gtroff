// publicaudit runs pinned verification tools and emits only candidate metadata.
// It is an optional audit program, not part of the parser or product CLI.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type candidate struct {
	Rule   string `json:"rule"`
	File   string `json:"file"`
	Line   int    `json:"line"`
	Commit string `json:"commit,omitempty"`
}
type receipt struct {
	Schema      int            `json:"schema"`
	Label       string         `json:"label"`
	Scanner     string         `json:"scanner"`
	Version     string         `json:"version"`
	Kind        string         `json:"kind"`
	ExitCode    int            `json:"scanner_exit_code"`
	Complete    bool           `json:"scan_complete"`
	Diagnostics map[string]int `json:"diagnostic_line_counts"`
	Candidates  []candidate    `json:"candidates"`
}

type result struct {
	stdout, stderr []byte
	code           int
}
type runner func(context.Context, string, []string) (result, error)

// No fields from scanner output are echoed in an error message.
var errIncomplete = errors.New("scan incomplete; inspect the receipt status and tool invocation locally")
var safeLabel = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,64}$`)
var safeRule = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.-]{0,127}$`)
var commitID = regexp.MustCompile(`^(?:[a-f0-9]{40}|[a-f0-9]{64})$`)

func relativeFile(root, name string) string {
	if filepath.IsAbs(name) {
		p, err := filepath.Rel(root, name)
		if err != nil {
			return "[outside source]"
		}
		name = p
	}
	p := filepath.Clean(name)
	if p == "." || p == ".." || strings.HasPrefix(p, ".."+string(filepath.Separator)) || strings.ContainsAny(p, "\x00\r\n") {
		return "[unreportable path]"
	}
	return filepath.ToSlash(p)
}

func redact(scanner, root string, raw []byte) ([]candidate, error) {
	rows := []candidate{}
	add := func(rule, file, commit string, line int) error {
		if !safeRule.MatchString(rule) || line < 0 || (commit != "" && !commitID.MatchString(commit)) {
			return errIncomplete
		}
		rows = append(rows, candidate{rule, relativeFile(root, file), line, commit})
		return nil
	}
	if scanner == "gitleaks" {
		// Decode only named fields. Match/Secret/context/author/message and
		// future scanner fields cannot pass through to the receipt.
		var findings []struct {
			RuleID, File, Commit string
			StartLine            int
		}
		if err := json.Unmarshal(raw, &findings); err != nil || findings == nil {
			return nil, errIncomplete
		}
		for _, f := range findings {
			if err := add(f.RuleID, f.File, f.Commit, f.StartLine); err != nil {
				return nil, err
			}
		}
		return rows, nil
	}
	s := bufio.NewScanner(bytes.NewReader(raw))
	s.Buffer(make([]byte, 4096), 16<<20)
	for s.Scan() {
		var f struct {
			DetectorName   string
			SourceMetadata struct {
				Data struct {
					Git struct {
						File, Commit string
						Line         int
					}
					Filesystem struct {
						File string
						Line int
					}
				}
			}
		}
		if err := json.Unmarshal(s.Bytes(), &f); err != nil {
			return nil, errIncomplete
		}
		g, fs := f.SourceMetadata.Data.Git, f.SourceMetadata.Data.Filesystem
		file, line := g.File, g.Line
		if file == "" {
			file, line = fs.File, fs.Line
		}
		if err := add(f.DetectorName, file, g.Commit, line); err != nil {
			return nil, err
		}
	}
	if s.Err() != nil {
		return nil, errIncomplete
	}
	return rows, nil
}

func diagnosticCounts(raw []byte) map[string]int {
	counts := make(map[string]int)
	for _, line := range bytes.Split(raw, []byte{'\n'}) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var entry struct{ Level string }
		if json.Unmarshal(line, &entry) != nil {
			counts["unstructured"]++
			continue
		}
		// Do not echo arbitrary level/message/error strings from a subprocess.
		switch entry.Level {
		case "info-0", "info", "warn", "error", "fatal":
			counts[entry.Level]++
		default:
			counts["other"]++
		}
	}
	return counts
}

type cappedBuffer struct {
	data      bytes.Buffer
	remaining int
	cancel    context.CancelFunc
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if len(p) > b.remaining {
		b.cancel()
		return 0, errIncomplete
	}
	b.remaining -= len(p)
	return b.data.Write(p)
}

func execute(ctx context.Context, binary string, args []string) (result, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	out := &cappedBuffer{remaining: 64 << 20, cancel: cancel}
	stderr := &cappedBuffer{remaining: 16 << 20, cancel: cancel}
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.WaitDelay = 5 * time.Second
	cmd.Stdout, cmd.Stderr = out, stderr
	for _, entry := range os.Environ() {
		if strings.HasPrefix(entry, "GITLEAKS_") || strings.HasPrefix(entry, "TRUFFLEHOG_") || strings.HasPrefix(entry, "GIT_") || strings.HasPrefix(entry, "GOMAXPROCS=") {
			continue
		}
		cmd.Env = append(cmd.Env, entry)
	}
	cmd.Env = append(cmd.Env, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull, "GOMAXPROCS=4")
	err := cmd.Run()
	r := result{out.data.Bytes(), stderr.data.Bytes(), 0}
	if ctx.Err() != nil {
		return r, errIncomplete
	}
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			return r, errIncomplete
		}
		r.code = exit.ExitCode()
	}
	return r, nil
}

func run(ctx context.Context, args []string, out io.Writer, invoke runner) error {
	flags := flag.NewFlagSet("publicaudit", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	scanner := flags.String("scanner", "", "gitleaks or trufflehog")
	binary := flags.String("executable", "", "explicit pinned scanner executable")
	kind := flags.String("kind", "", "git (controlled bare mirror) or filesystem")
	source := flags.String("source", "", "local input directory")
	label := flags.String("label", "", "public scope label")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 || *binary == "" || *source == "" || !safeLabel.MatchString(*label) || (*kind != "git" && *kind != "filesystem") || (*scanner != "gitleaks" && *scanner != "trufflehog") {
		return errors.New("usage: publicaudit -scanner gitleaks|trufflehog -executable FILE -kind git|filesystem -source DIRECTORY -label NAME")
	}
	root, err := filepath.Abs(*source)
	if err != nil {
		return errors.New("cannot resolve input directory")
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		return errors.New("source must be an existing local directory")
	}
	tool, err := filepath.Abs(*binary)
	if err != nil {
		return errors.New("cannot resolve scanner executable")
	}
	version, versionArgs, expected := "8.30.1", []string{"version"}, "8.30.1"
	if *scanner == "trufflehog" {
		version, versionArgs, expected = "3.99.9", []string{"--version"}, "trufflehog 3.99.9"
	}
	v, err := invoke(ctx, tool, versionArgs)
	if err != nil || v.code != 0 || strings.TrimSpace(string(v.stdout)) != expected {
		return errors.New("scanner version does not match the pinned audit version")
	}
	dir, err := os.MkdirTemp("", "gtroff-publicaudit-")
	if err != nil {
		return errors.New("cannot create private audit configuration")
	}
	defer os.RemoveAll(dir)
	config, ignore := filepath.Join(dir, "builtin.toml"), filepath.Join(dir, "empty.ignore")
	if err := os.WriteFile(config, []byte("[extend]\nuseDefault = true\n"), 0600); err != nil {
		return errors.New("cannot write audit configuration")
	}
	if err := os.WriteFile(ignore, nil, 0600); err != nil {
		return errors.New("cannot write empty ignore file")
	}
	var command []string
	if *scanner == "gitleaks" {
		mode := "dir"
		if *kind == "git" {
			mode = "git"
		}
		command = []string{mode, root, "--config", config, "--no-banner", "--no-color", "--redact=100", "--log-level=error", "--ignore-gitleaks-allow", "--gitleaks-ignore-path", ignore, "--report-format=json", "--report-path=-", "--exit-code=10"}
		if *kind == "git" {
			command = append(command, "--log-opts=--all --full-history --diff-merges=separate --no-ext-diff --no-textconv")
		}
	} else {
		command = []string{*kind, root, "--no-verification", "--no-update", "--no-ignore-tag", "--results=verified,unverified,unknown,filtered_unverified", "--concurrency=4", "--archive-max-size=256MB", "--archive-max-depth=4", "--archive-timeout=30s", "--fail", "--fail-on-scan-errors", "--json"}
		if *kind == "git" {
			path := filepath.ToSlash(root)
			if filepath.VolumeName(root) != "" {
				path = "/" + path
			}
			command[1] = (&url.URL{Scheme: "file", Path: path}).String()
			command = append(command, "--bare", "--trust-local-git-config")
		}
	}
	raw, scanErr := invoke(ctx, tool, command)
	rows, decodeErr := redact(*scanner, root, raw.stdout)
	r := receipt{1, *label, *scanner, version, *kind, raw.code, false, diagnosticCounts(raw.stderr), rows}
	validExit := raw.code == 0
	if len(rows) > 0 {
		validExit = (*scanner == "gitleaks" && raw.code == 10) || (*scanner == "trufflehog" && raw.code == 183)
	}
	r.Complete = scanErr == nil && decodeErr == nil && validExit && r.Diagnostics["error"] == 0 && r.Diagnostics["fatal"] == 0
	for level, count := range r.Diagnostics {
		if count > 0 && level != "info" && level != "info-0" {
			r.Complete = false
		}
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(r); err != nil {
		return errors.New("cannot write audit receipt")
	}
	if !r.Complete {
		return errIncomplete
	}
	return nil
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	if err := run(ctx, os.Args[1:], os.Stdout, execute); err != nil {
		fmt.Fprintln(os.Stderr, "publicaudit:", err)
		os.Exit(1)
	}
}
