package manlib_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/arran4/golang-gtroff/manlib"
)

func TestSystemSearchExplicitProbeAndArguments(t *testing.T) {
	var probes []string
	called := 0
	provider, e := manlib.DetectSystemSearch(func(name string) (string, error) {
		probes = append(probes, name)
		if name == "apropos" {
			return "/explicit/apropos", nil
		}
		return "", errors.New("absent")
	}, func(ctx context.Context, program string, args []string, max int) ([]byte, error) {
		called++
		if program != "/explicit/apropos" || !reflect.DeepEqual(args, []string{"--", "-option ; $(command)"}) || max != 1024 {
			t.Fatalf("unsafe command boundary %s %#v %d", program, args, max)
		}
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("missing deadline")
		}
		return []byte("tool (1) - <script>label</script>\nodd output line\n"), nil
	}, time.Second, 1024)
	if e != nil || called != 0 || !reflect.DeepEqual(probes, []string{"apropos"}) {
		t.Fatal(probes, e)
	}
	hits, e := provider.Search(context.Background(), "-option ; $(command)")
	if e != nil || len(hits) != 2 || hits[0].Description != "<script>label</script>" || hits[1].Description != "odd output line" {
		t.Fatal(hits, e)
	}
	if _, e := manlib.DetectSystemSearch(func(string) (string, error) { return "", errors.New("absent") }, manlib.RunCommand, time.Second, 1024); !errors.Is(e, manlib.ErrUnavailable) {
		t.Fatal(e)
	}
}
func TestSystemSearchTimeoutAndOutputLimit(t *testing.T) {
	p, e := manlib.DetectSystemSearch(func(string) (string, error) { return "program", nil }, func(ctx context.Context, _ string, _ []string, _ int) ([]byte, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}, time.Millisecond, 32)
	if e != nil {
		t.Fatal(e)
	}
	if _, e := p.Search(context.Background(), "query"); !errors.Is(e, context.DeadlineExceeded) {
		t.Fatal(e)
	}
	p, e = manlib.DetectSystemSearch(func(string) (string, error) { return "program", nil }, func(context.Context, string, []string, int) ([]byte, error) {
		return []byte(strings.Repeat("x", 64)), nil
	}, time.Second, 32)
	if e != nil {
		t.Fatal(e)
	}
	if _, e := p.Search(context.Background(), "query"); !errors.Is(e, manlib.ErrLimit) {
		t.Fatal(e)
	}
}
func TestRunCommandProcessBoundary(t *testing.T) {
	executable, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	raw, e := manlib.RunCommand(ctx, executable, []string{"-test.run=^TestSystemProcessHelper$", "--", "manlib-helper", "arguments", "; $(never-a-shell)"}, 4096)
	if e != nil {
		t.Fatal(e)
	}
	var args []string
	if e = json.Unmarshal(raw, &args); e != nil || len(args) != 1 || args[0] != "; $(never-a-shell)" {
		t.Fatal(string(raw), e)
	}
	if _, e := manlib.RunCommand(ctx, executable, []string{"-test.run=^TestSystemProcessHelper$", "--", "manlib-helper", "overflow"}, 32); e == nil {
		t.Fatal("process output escaped bound")
	}
}
func TestSystemProcessHelper(t *testing.T) {
	for i, s := range os.Args {
		if s != "manlib-helper" {
			continue
		}
		switch os.Args[i+1] {
		case "arguments":
			_ = json.NewEncoder(os.Stdout).Encode(os.Args[i+2:])
		case "overflow":
			_, _ = os.Stdout.WriteString(strings.Repeat("x", 32768))
		}
		os.Exit(0)
	}
}
