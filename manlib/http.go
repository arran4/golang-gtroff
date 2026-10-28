package manlib

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/renderhtml"
)

const contentPolicy = "default-src 'none'; font-src data:; style-src 'unsafe-inline'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'"

type HandlerOptions struct {
	Render                                      renderhtml.Options
	SearchProvider                              SearchProvider
	Notices                                     []string
	AllowedHosts                                []string // default localhost/loopback; explicit "*" allows all hosts
	MaxConcurrent, MaxResponseBytes, MaxResults int
}
type libraryHandler struct {
	index    *Index
	o        HandlerOptions
	gate     chan struct{}
	template *template.Template
}

// Handler serves only an immutable index. Request data never selects filesystem
// paths, source includes, fonts, remote URLs or a program name.
func Handler(index *Index, o HandlerOptions) (http.Handler, error) {
	if index == nil {
		return nil, fmt.Errorf("nil man index")
	}
	if o.MaxConcurrent == 0 {
		o.MaxConcurrent = 4
	}
	if o.MaxResponseBytes == 0 {
		o.MaxResponseBytes = 4 << 20
	}
	if o.MaxResults == 0 {
		o.MaxResults = 1000
	}
	if o.MaxConcurrent < 1 || o.MaxConcurrent > 32 || o.MaxResponseBytes < 1024 || o.MaxResponseBytes > 16<<20 || o.MaxResults < 1 || o.MaxResults > 1000 {
		return nil, fmt.Errorf("HTTP budgets exceed supported bounds")
	}
	if o.Render.Node != 0 || o.Render.Section != "" || o.Render.Fragment || o.Render.NoAnchors {
		return nil, fmt.Errorf("HTTP pages require complete anchored documents; use request heading navigation")
	}
	if err := renderhtml.ValidateOptions(o.Render); err != nil {
		return nil, err
	}
	o.Render.Fragment = true
	if o.Render.MaxOutputBytes <= 0 || o.Render.MaxOutputBytes > o.MaxResponseBytes {
		o.Render.MaxOutputBytes = o.MaxResponseBytes
	}
	o.AllowedHosts = append([]string(nil), o.AllowedHosts...)
	o.Notices = append([]string(nil), o.Notices...)
	// Keep caller mutation from changing fonts while requests are in flight.
	o.Render.Fonts.Regular = bytes.Clone(o.Render.Fonts.Regular)
	o.Render.Fonts.Bold = bytes.Clone(o.Render.Fonts.Bold)
	o.Render.Fonts.Italic = bytes.Clone(o.Render.Fonts.Italic)
	o.Render.Fonts.Mono = bytes.Clone(o.Render.Fonts.Mono)
	t, err := template.New("library").Parse(libraryTemplate)
	if err != nil {
		return nil, err
	}
	return &libraryHandler{index: index, o: o, gate: make(chan struct{}, o.MaxConcurrent), template: t}, nil
}

type heading struct {
	Text, URL  string
	Subsection bool
}
type view struct {
	Title, Query, Summary, Previous, Next string
	Hits                                  []Hit
	Notices                               []string
	Collection                            []Notice
	Headings                              []heading
	Rendered                              template.HTML
}

func (h *libraryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Security-Policy", contentPolicy)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != "GET" && r.Method != "HEAD" {
		w.Header().Set("Allow", "GET, HEAD")
		h.error(w, r, 405, "read-only library; use GET or HEAD")
		return
	}
	if !h.allowedHost(r.Host) {
		h.error(w, r, 403, "unrecognized library host")
		return
	}
	if len(r.RequestURI) > 4096 || len(r.URL.RawQuery) > 2048 {
		h.error(w, r, 414, "request URI is too long")
		return
	}
	if r.URL.Path == "" {
		r.URL.Path = "/"
	}
	escaped := strings.ToLower(r.URL.EscapedPath())
	if strings.ContainsAny(r.URL.Path, "\\\x00") || path.Clean(r.URL.Path) != r.URL.Path || strings.Contains(escaped, "%2f") || strings.Contains(escaped, "%5c") {
		h.error(w, r, 400, "invalid library path")
		return
	}
	if err := r.Context().Err(); err != nil {
		h.error(w, r, 408, "request cancelled")
		return
	}
	select {
	case h.gate <- struct{}{}:
		defer func() { <-h.gate }()
	default:
		w.Header().Set("Retry-After", "1")
		h.error(w, r, 503, "library is busy")
		return
	}
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		h.error(w, r, 400, "invalid query encoding")
		return
	}
	for name, values := range q {
		if len(values) != 1 || !(name == "q" || name == "page" || name == "root" || name == "heading") {
			h.error(w, r, 400, "unknown or repeated query parameter")
			return
		}
	}
	v := view{Title: "Man library", Notices: append([]string(nil), h.o.Notices...), Collection: h.index.Notices()}
	status := 200
	switch {
	case r.URL.Path == "/" || r.URL.Path == "/search":
		if q.Get("root") != "" || q.Get("heading") != "" {
			h.error(w, r, 400, "page-only query parameter")
			return
		}
		query := q.Get("q")
		if !validQuery(query) {
			h.error(w, r, 400, "search query exceeds supported bounds")
			return
		}
		v.Query = query
		var hits []Hit
		var err error
		if strings.TrimSpace(query) == "" {
			hits = make([]Hit, 0, len(h.index.pages))
			for _, p := range h.index.pages {
				if e := r.Context().Err(); e != nil {
					err = e
					break
				}
				entry := p.entry
				hits = append(hits, Hit{Name: entry.Name, Section: entry.Section, Description: entry.Description, Root: entry.Root, URL: entry.URL})
			}
		} else {
			hits, err = h.index.Search(r.Context(), query, h.o.MaxResults)
		}
		if err != nil {
			h.error(w, r, 408, "search cancelled")
			return
		}
		if strings.TrimSpace(query) != "" && len(hits) == h.o.MaxResults {
			v.Notices = append(v.Notices, "Search is limited to the first "+strconv.Itoa(h.o.MaxResults)+" matches; refine the query.")
		}
		if strings.TrimSpace(query) != "" && h.o.SearchProvider != nil {
			external, e := h.o.SearchProvider.Search(r.Context(), query)
			if e != nil {
				v.Notices = append(v.Notices, "System search unavailable; built-in results shown.")
			} else {
				seen := map[string]bool{}
				for _, hit := range hits {
					seen[pageKey(hit.Name, hit.Section)] = true
				}
				totalExternalBytes := 0
				for n, hit := range external {
					totalExternalBytes += len(hit.Name) + len(hit.Section) + len(hit.Description)
					if totalExternalBytes > 64<<10 {
						v.Notices = append(v.Notices, "System result byte budget exceeded; remaining results omitted.")
						break
					}
					if n >= 256 {
						v.Notices = append(v.Notices, "System results limited to 256 entries.")
						break
					}
					if err := r.Context().Err(); err != nil {
						h.error(w, r, 408, "request cancelled")
						return
					}
					hit.External = true
					hit.URL = ""
					hit.Root = "system"
					if seen[pageKey(hit.Name, hit.Section)] && hit.Name != "" {
						continue
					}
					if p := h.index.find(hit.Name, hit.Section, ""); p != nil {
						hit.URL = p.entry.URL
					}
					seen[pageKey(hit.Name, hit.Section)] = true
					hits = append(hits, hit)
				}
			}
		}
		pageNumber := 1
		if s := q.Get("page"); s != "" {
			n, e := strconv.Atoi(s)
			if e != nil || n < 1 || n > 1000 {
				h.error(w, r, 400, "invalid result page")
				return
			}
			pageNumber = n
		}
		first := (pageNumber - 1) * 50
		if first >= len(hits) && pageNumber != 1 {
			h.error(w, r, 404, "result page not found")
			return
		}
		last := min(first+50, len(hits))
		v.Hits = hits[first:last]
		v.Summary = fmt.Sprintf("%d matches", len(hits))
		if len(hits) == 0 {
			v.Summary = "No matching pages."
		}
		if pageNumber > 1 {
			v.Previous = resultsURL(r.URL.Path, query, pageNumber-1)
		}
		if last < len(hits) {
			v.Next = resultsURL(r.URL.Path, query, pageNumber+1)
		}
	case strings.HasPrefix(r.URL.Path, "/page/"):
		if q.Get("q") != "" || q.Get("page") != "" {
			h.error(w, r, 400, "search-only query parameter")
			return
		}
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/page/"), "/")
		if len(parts) != 2 || !sectionPattern.MatchString(parts[0]) || !validName(parts[1]) {
			h.error(w, r, 400, "invalid page identity")
			return
		}
		p := h.index.find(parts[1], parts[0], q.Get("root"))
		if p == nil {
			h.error(w, r, 404, "page not found")
			return
		}
		doc, parseErr := h.index.Document(r.Context(), p.entry.Name, p.entry.Section, p.entry.Root)
		if parseErr != nil {
			if r.Context().Err() != nil {
				h.error(w, r, 408, "request cancelled")
				return
			}
			h.error(w, r, 503, "source snapshot could not be parsed")
			return
		}
		o := h.o.Render
		o.Section = q.Get("heading")
		if len(o.Section) > 512 {
			h.error(w, r, 400, "heading selector is too long")
			return
		}
		if o.Section != "" {
			if _, e := doc.SelectSection(o.Section, false); e != nil {
				h.error(w, r, 404, "heading not found")
				return
			}
		}
		var out bytes.Buffer
		_, e := renderhtml.Render(&out, doc, o)
		if e != nil {
			h.error(w, r, 422, "page cannot be rendered under the configured strictness or output budget")
			return
		}
		v.Title = p.entry.Name + "(" + p.entry.Section + ")"
		for _, n := range doc.Nodes {
			if n.Kind == man.Section || n.Kind == man.Subsection {
				v.Headings = append(v.Headings, heading{Text: n.Heading, URL: p.entry.URL + "#man-node-" + strconv.FormatUint(uint64(n.ID), 10), Subsection: n.Kind == man.Subsection})
			}
		}
		// This HTML comes exclusively from the validated escaping renderer above,
		// never from source bytes, filenames, query text or external provider output.
		v.Rendered = template.HTML(out.String())
	default:
		h.error(w, r, 404, "route not found")
		return
	}
	if r.Context().Err() != nil {
		h.error(w, r, 408, "request cancelled")
		return
	}
	out := &limitedHTML{max: h.o.MaxResponseBytes}
	if err := h.template.Execute(out, v); err != nil {
		h.error(w, r, 503, "response budget exceeded")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(out.Len()))
	w.WriteHeader(status)
	if r.Method != "HEAD" {
		_, _ = w.Write(out.Bytes())
	}
}
func resultsURL(route, query string, page int) string {
	q := url.Values{}
	if query != "" {
		q.Set("q", query)
	}
	q.Set("page", strconv.Itoa(page))
	return route + "?" + q.Encode()
}
func (h *libraryHandler) allowedHost(host string) bool {
	name := host
	if v, _, e := net.SplitHostPort(host); e == nil {
		name = v
	}
	name = strings.Trim(name, "[]")
	for _, allowed := range h.o.AllowedHosts {
		if allowed == "*" || strings.EqualFold(allowed, name) {
			return true
		}
	}
	if len(h.o.AllowedHosts) > 0 {
		return false
	}
	if strings.EqualFold(name, "localhost") {
		return true
	}
	ip, e := netip.ParseAddr(name)
	return e == nil && ip.IsLoopback()
}
func (h *libraryHandler) error(w http.ResponseWriter, r *http.Request, status int, message string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(message)+1))
	w.WriteHeader(status)
	if r.Method != "HEAD" {
		_, _ = fmt.Fprintln(w, message)
	}
}

type limitedHTML struct {
	bytes.Buffer
	max int
}

func (w *limitedHTML) WriteString(s string) (int, error) { return w.Write([]byte(s)) }
func (w *limitedHTML) Write(b []byte) (int, error) {
	if len(b) > w.max-w.Len() {
		return 0, ErrLimit
	}
	return w.Buffer.Write(b)
}

// Serve runs an owned listener with HTTP bounds and graceful context shutdown.
// It uses no global mux, signal handler or process streams.
func Serve(ctx context.Context, listener net.Listener, handler http.Handler) error {
	if ctx == nil || listener == nil || handler == nil {
		return fmt.Errorf("nil server capability")
	}
	server := &http.Server{Handler: handler, ErrorLog: log.New(io.Discard, "", 0), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 << 10}
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if e := server.Shutdown(shutdown); e != nil {
				_ = server.Close()
			}
		case <-done:
		}
	}()
	err := server.Serve(listener)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}
