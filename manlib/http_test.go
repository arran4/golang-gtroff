package manlib_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/arran4/golang-gtroff/manlib"
	"github.com/arran4/golang-gtroff/renderhtml"
)

func handler(t *testing.T) http.Handler {
	t.Helper()
	h, e := manlib.Handler(fixtureIndex(t), manlib.HandlerOptions{Render: renderhtml.Options{Assets: renderhtml.OmitFonts}})
	if e != nil {
		t.Fatal(e)
	}
	return h
}
func TestRealHTTPAndAnchors(t *testing.T) {
	s := httptest.NewServer(handler(t))
	defer s.Close()
	for _, path := range []string{"/", "/search?q=utility", "/page/1/tool", "/page/3/tool", "/page/3ssl/ssl"} {
		res, e := s.Client().Get(s.URL + path)
		if e != nil {
			t.Fatal(e)
		}
		body, e := io.ReadAll(res.Body)
		_ = res.Body.Close()
		if e != nil || res.StatusCode != 200 {
			t.Fatalf("%s %d %s %v", path, res.StatusCode, body, e)
		}
		if !strings.Contains(res.Header.Get("Content-Security-Policy"), "default-src 'none'") {
			t.Fatal("missing CSP")
		}
		if path == "/page/1/tool" && (!strings.Contains(string(body), "#man-node-2") || !strings.Contains(string(body), `id="man-node-2"`)) {
			t.Fatal("navigation lacks actual heading anchors")
		}
	}
	req, _ := http.NewRequest("HEAD", s.URL+"/page/1/tool", nil)
	res, e := s.Client().Do(req)
	if e != nil {
		t.Fatal(e)
	}
	body, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if len(body) > 0 || res.ContentLength < 1 {
		t.Fatal("HEAD response")
	}
}
func TestTraversalHostAndMethods(t *testing.T) {
	h := handler(t)
	for _, tc := range []struct {
		method, path, host string
		status             int
	}{{"POST", "/", "localhost", 405}, {"GET", "/page/1/..", "localhost", 400}, {"GET", "/page/1/%2e%2e", "localhost", 400}, {"GET", "/page/1/a%2fb", "localhost", 400}, {"GET", "/page/1/a%5cb", "localhost", 400}, {"GET", "//page/1/tool", "localhost", 400}, {"GET", "/", "evil.example", 403}, {"GET", "/page/1/absent", "localhost", 404}, {"GET", "/search?q=x&q=y", "localhost", 400}, {"GET", "/?file=/etc/passwd", "localhost", 400}, {"GET", "/search?page=-1", "localhost", 400}, {"GET", "/search?q=%00", "localhost", 400}} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.Host = tc.host
		out := httptest.NewRecorder()
		h.ServeHTTP(out, req)
		if out.Code != tc.status {
			t.Fatalf("%s %s host %s: %d %s", tc.method, tc.path, tc.host, out.Code, out.Body.String())
		}
	}
}
func TestServeCancellation(t *testing.T) {
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	h := handler(t)
	go func() { done <- manlib.Serve(ctx, l, h) }()
	c := &http.Client{Timeout: time.Second, Transport: &http.Transport{}}
	res, e := c.Get("http://" + l.Addr().String() + "/")
	if e != nil {
		t.Fatal(e)
	}
	_ = res.Body.Close()
	cancel()
	select {
	case e := <-done:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop")
	}
}

type waitingProvider struct {
	started chan struct{}
	release chan struct{}
}

func (p waitingProvider) Search(ctx context.Context, _ string) ([]manlib.Hit, error) {
	close(p.started)
	select {
	case <-p.release:
		return nil, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
func TestRequestConcurrencyAndCancellation(t *testing.T) {
	p := waitingProvider{make(chan struct{}), make(chan struct{})}
	h, e := manlib.Handler(fixtureIndex(t), manlib.HandlerOptions{SearchProvider: p, MaxConcurrent: 1, Render: renderhtml.Options{Assets: renderhtml.OmitFonts}})
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/search?q=tool", nil).WithContext(ctx)
	req.Host = "localhost"
	first := httptest.NewRecorder()
	done := make(chan struct{})
	go func() { h.ServeHTTP(first, req); close(done) }()
	select {
	case <-p.started:
	case <-time.After(time.Second):
		t.Fatal("provider did not start")
	}
	secondReq := httptest.NewRequest("GET", "/", nil)
	secondReq.Host = "localhost"
	second := httptest.NewRecorder()
	h.ServeHTTP(second, secondReq)
	if second.Code != 503 {
		t.Fatal("concurrency gate not enforced", second.Code)
	}
	cancel()
	select {
	case <-done:
		if first.Code != 408 {
			t.Fatal("cancellation did not stop response", first.Code)
		}
	case <-time.After(time.Second):
		t.Fatal("request cancellation stalled")
	}
}
func TestResponseBudgetAndExplicitHost(t *testing.T) {
	h, e := manlib.Handler(fixtureIndex(t), manlib.HandlerOptions{MaxResponseBytes: 1024, AllowedHosts: []string{"man.example"}, Render: renderhtml.Options{Assets: renderhtml.OmitFonts}})
	if e != nil {
		t.Fatal(e)
	}
	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "man.example"
	out := httptest.NewRecorder()
	h.ServeHTTP(out, req)
	if out.Code != 503 || strings.Contains(out.Body.String(), "<!doctype") {
		t.Fatal("response budget emitted partial HTML", out.Code, out.Body.String())
	}
}
func FuzzLibraryHTTP(f *testing.F) {
	for _, s := range []string{"/", "/search?q=tool", "/page/1/tool", "/page/1/%2e%2e", "/?q=%00"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, target string) {
		if len(target) > 4096 || !strings.HasPrefix(target, "/") || strings.ContainsAny(target, "\r\n\x00") {
			t.Skip()
		}
		u, e := url.ParseRequestURI(target)
		if e != nil {
			t.Skip()
		}
		req := httptest.NewRequest("GET", "http://localhost/", nil)
		req.URL = u
		req.RequestURI = target
		req.Host = "localhost"
		out := httptest.NewRecorder()
		handler(t).ServeHTTP(out, req)
		if out.Body.Len() > 4<<20 {
			t.Fatal("response budget escaped")
		}
		if out.Code < 200 || out.Code > 599 {
			t.Fatal("invalid status")
		}
	})
}

func TestCataloguePaginationIndependentOfSearchCap(t *testing.T) {
	h, e := manlib.Handler(fixtureIndex(t), manlib.HandlerOptions{MaxResults: 1, Render: renderhtml.Options{Assets: renderhtml.OmitFonts}})
	if e != nil {
		t.Fatal(e)
	}
	out := httptest.NewRecorder()
	h.ServeHTTP(out, httptest.NewRequest("GET", "http://localhost/", nil))
	if out.Code != 200 || !strings.Contains(out.Body.String(), "3 matches") {
		t.Fatal("catalogue was truncated by search cap", out.Code, out.Body.String())
	}
}
