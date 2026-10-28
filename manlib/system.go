package manlib

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var ErrUnavailable = errors.New("optional system man search unavailable")

// CommandRunner executes an explicit program/argument array with a context and
// bounded stdout. DetectSystemSearch never invokes it until Search is called.
type CommandRunner func(context.Context, string, []string, int) ([]byte, error)
type systemSearch struct {
	program, kind string
	run           CommandRunner
	timeout       time.Duration
	maxBytes      int
}

// DetectSystemSearch probes only explicitly supplied lookup/execution capabilities.
// nil capabilities or missing tools return ErrUnavailable; callers retain built-in search.
func DetectSystemSearch(lookPath func(string) (string, error), run CommandRunner, timeout time.Duration, maxBytes int) (SearchProvider, error) {
	if lookPath == nil || run == nil {
		return nil, ErrUnavailable
	}
	if timeout == 0 {
		timeout = time.Second
	}
	if maxBytes == 0 {
		maxBytes = 64 << 10
	}
	if timeout < time.Millisecond || timeout > 5*time.Second || maxBytes < 1 || maxBytes > 256<<10 {
		return nil, fmt.Errorf("system search limits exceed supported bounds")
	}
	for _, name := range []string{"apropos", "whatis", "man"} {
		program, err := lookPath(name)
		if err == nil && program != "" {
			return &systemSearch{program, name, run, timeout, maxBytes}, nil
		}
	}
	return nil, ErrUnavailable
}

var systemLine = regexp.MustCompile(`^(.+?)\s+\(([^()\s]+)\)\s+-\s*(.*)$`)

func (s *systemSearch) Search(ctx context.Context, query string) ([]Hit, error) {
	if ctx == nil || !validQuery(query) || strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("invalid system search context/query")
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	args := []string{"--", query}
	if s.kind == "man" {
		args = []string{"-k", "--", query}
	}
	raw, err := s.run(ctx, s.program, args, s.maxBytes)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(raw) > s.maxBytes {
		return nil, ErrLimit
	}
	var hits []Hit
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if len(hits) >= 256 {
			return hits, ErrLimit
		}
		m := systemLine.FindStringSubmatch(line)
		if m == nil {
			hits = append(hits, Hit{Description: line, External: true})
			continue
		}
		for _, name := range strings.Split(m[1], ",") {
			name = strings.TrimSpace(name)
			if len(hits) >= 256 {
				return hits, ErrLimit
			}
			if validName(name) && sectionPattern.MatchString(m[2]) {
				hits = append(hits, Hit{Name: name, Section: m[2], Description: m[3], External: true})
			} else {
				hits = append(hits, Hit{Description: line, External: true})
			}
		}
	}
	return hits, nil
}

type boundedOutput struct {
	data  []byte
	limit int
}

func (w *boundedOutput) Write(b []byte) (int, error) {
	if len(b) > w.limit-len(w.data) {
		return 0, ErrLimit
	}
	w.data = append(w.data, b...)
	return len(b), nil
}

// RunCommand is the optional real process boundary. It uses no shell, discards
// stderr and bounds stdout. Call through DetectSystemSearch for a timeout.
func RunCommand(ctx context.Context, program string, args []string, maxBytes int) ([]byte, error) {
	if ctx == nil || maxBytes < 1 || maxBytes > 256<<10 {
		return nil, fmt.Errorf("invalid process context/output budget")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out := &boundedOutput{limit: maxBytes}
	cmd := exec.CommandContext(ctx, program, args...)
	cmd.Stdout = out
	cmd.Stderr = io.Discard
	cmd.WaitDelay = 100 * time.Millisecond
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return out.data, nil
}
