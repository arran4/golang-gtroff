package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/arran4/golang-gtroff/manlib"
)

type closeCallback func() error

func (c closeCallback) Close() error { return c() }
func TestServeCLIWithExplicitCapabilities(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := make(chan string, 1)
	done := make(chan int, 1)
	closed := false
	var out, diags bytes.Buffer
	e := Environment{Context: ctx, Out: &out, Err: &diags, OpenRoot: func(path string) (manlib.Root, io.Closer, error) {
		if path != "collection" {
			t.Error("unexpected root", path)
		}
		return manlib.Root{FS: fstest.MapFS{"man1/tool.1": {Data: []byte(".TH TOOL 1\n.SH NAME\ntool \\- searchable helper\n")}}}, closeCallback(func() error { closed = true; return nil }), nil
	}, Listen: func(address string) (net.Listener, error) {
		if !closed {
			t.Error("root still open before listening")
		}
		if address != "127.0.0.1:0" {
			t.Error("unexpected binding", address)
		}
		l, e := net.Listen("tcp", address)
		if e == nil {
			ready <- l.Addr().String()
		}
		return l, e
	}, SystemSearch: func() (manlib.SearchProvider, error) {
		t.Error("disabled provider probed")
		return nil, manlib.ErrUnavailable
	}}
	go func() {
		done <- Run([]string{"serve", "--root", "collection", "--listen", "127.0.0.1:0", "--assets", "omit-fonts"}, e)
	}()
	var address string
	select {
	case address = <-ready:
	case <-time.After(2 * time.Second):
		t.Fatal("no listener")
	}
	client := &http.Client{Timeout: time.Second, Transport: &http.Transport{}}
	res, err := client.Get("http://" + address + "/search?q=searchable")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if res.StatusCode != 200 || !bytes.Contains(body, []byte("tool(1)")) {
		t.Fatal(res.StatusCode, string(body))
	}
	cancel()
	select {
	case code := <-done:
		if code != 0 || out.Len() != 0 || !strings.Contains(diags.String(), "serving 1 pages") {
			t.Fatalf("serve exit %d %s", code, diags.String())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("CLI did not shut down")
	}
}
func TestServeUsageBeforeAccess(t *testing.T) {
	for _, args := range [][]string{{"serve"}, {"serve", "--root", "x", "--listen", "0.0.0.0:8080"}, {"serve", "--root", "x", "--listen", "127.0.0.1:bad"}, {"serve", "--root", "x", "--max-pages", "0"}, {"serve", "--root", "x", "--max-total-bytes", "999999999"}, {"serve", "--root", "x", "--assets", "remote"}, {"serve", "--root", "x", "file"}} {
		code := Run(args, Environment{OpenRoot: func(string) (manlib.Root, io.Closer, error) {
			t.Fatal("opened on usage failure")
			return manlib.Root{}, nil, nil
		}})
		if code != ExitUsage {
			t.Fatal(args, code)
		}
	}
}
func TestServeBudgetClosesRootsBeforeFailure(t *testing.T) {
	closed := false
	listened := false
	var diags bytes.Buffer
	e := Environment{Err: &diags, OpenRoot: func(string) (manlib.Root, io.Closer, error) {
		return manlib.Root{FS: fstest.MapFS{"large.1": {Data: []byte("far beyond the cap")}}}, closeCallback(func() error { closed = true; return nil }), nil
	}, Listen: func(string) (net.Listener, error) { listened = true; return nil, errors.New("not expected") }}
	if code := Run([]string{"serve", "--root", "x", "--max-file-bytes", "2"}, e); code != ExitIO || !closed || listened {
		t.Fatal(code, closed, listened, diags.String())
	}
}

func TestServeTotalBudgetStopsBeforeListen(t *testing.T) {
	closed := 0
	var diags bytes.Buffer
	e := Environment{Err: &diags, OpenRoot: func(name string) (manlib.Root, io.Closer, error) {
		sys := fstest.MapFS{"good.1": {Data: []byte("accepted\n")}}
		if name == "bad" {
			sys = fstest.MapFS{"bad.1.gz": {Data: []byte("not gzip\n")}}
		}
		return manlib.Root{FS: sys}, closeCallback(func() error { closed++; return nil }), nil
	}, Listen: func(string) (net.Listener, error) {
		t.Fatal("listened after failed build exceeded total budget")
		return nil, nil
	}}
	code := Run([]string{"serve", "--root", "good", "--root", "bad", "--max-total-bytes", "12"}, e)
	if code != ExitIO || closed != 2 || !strings.Contains(diags.String(), manlib.ErrLimit.Error()) {
		t.Fatalf("code=%d closed=%d diagnostics=%s", code, closed, diags.String())
	}
}
