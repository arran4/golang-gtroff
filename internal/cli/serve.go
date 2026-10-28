package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/netip"
	"strconv"
	"strings"

	"github.com/arran4/golang-gtroff/cmd"
	"github.com/arran4/golang-gtroff/manlib"
	"github.com/arran4/golang-gtroff/renderhtml"
)

const serveHelp = `Usage: gtroff serve --root DIR [--root DIR ...] [OPTIONS]

Build a read-only man library from explicit directories and serve its snapshot.
Plain .1/.3p/etc and .gz pages are supported. No host man directories are scanned
implicitly. Includes, scripts, remote resources and arbitrary request paths are
never opened. Restart to refresh source changes.

  --listen ADDRESS        Default: 127.0.0.1:8080; port 0 chooses a free port
  --root DIR              Required, repeatable; configured order breaks ties
  --system-search         Explicitly probe apropos/whatis/man -k; optional fallback
  --allow-remote          Permit non-loopback binding and arbitrary HTTP Host values
  --max-pages COUNT       Index cap, default 10000, ceiling 50000
  --max-total-bytes COUNT  Source/decode budget, default 67108864, ceiling 268435456
  --max-file-bytes COUNT   Raw/decoded per-page cap, default 2097152, ceiling 8388608
  --theme light|dark      Page theme, default light
  --assets embed|omit-fonts  Default: embed (bundled Go fonts)

Search matches names, sections, NAME descriptions and full semantic text.
System search only supplements indexed content and never opens its page paths.
The listener is localhost-only by default. Remote mode has no authentication:
serve only collections intended for every client able to reach that listener.
Press Ctrl-C to stop. Listening address/status go to stderr; stdout is unused.
Exit codes: 0 stopped normally, 2 usage, 3 filesystem/listener/resource failure.
`

type rootFlags []string

func (r *rootFlags) String() string { return strings.Join(*r, ",") }
func (r *rootFlags) Set(s string) error {
	if s == "" {
		return fmt.Errorf("root cannot be empty")
	}
	*r = append(*r, s)
	return nil
}
// Serve is a subcommand `gtroff serve` Browse and search explicit man-page directories over HTTP
//
// Usage: gtroff serve --root DIR [--root DIR ...] [OPTIONS]
//
// Build a read-only man library from explicit directories and serve its snapshot.
// Plain .1/.3p/etc and .gz pages are supported. No host man directories are scanned
// implicitly. Includes, scripts, remote resources and arbitrary request paths are
// never opened. Restart to refresh source changes.
//
// Flags:
//
//	listen: --listen (default: "127.0.0.1:8080") Default: 127.0.0.1:8080; port 0 chooses a free port
//	root: --root (default: nil) Required, repeatable; configured order breaks ties
//	systemSearch: --system-search (default: false) Explicitly probe apropos/whatis/man -k; optional fallback
//	allowRemote: --allow-remote (default: false) Permit non-loopback binding and arbitrary HTTP Host values
//	maxPages: --max-pages (default: 10000) Index cap, default 10000, ceiling 50000
//	maxTotalBytes: --max-total-bytes (default: 67108864) Source/decode budget, default 67108864, ceiling 268435456
//	maxFileBytes: --max-file-bytes (default: 2097152) Raw/decoded per-page cap, default 2097152, ceiling 8388608
//	theme: --theme (default: "light") Page theme, default light
//	assets: --assets (default: "embed") Default: embed (bundled Go fonts)
func Serve(listen string, root []string, systemSearch, allowRemote bool, maxPages, maxTotalBytes, maxFileBytes int, theme, assets string) error {
	code := runServe(listen, root, systemSearch, allowRemote, maxPages, maxTotalBytes, maxFileBytes, theme, assets, getEnv())
	if code != ExitOK {
		return &cmd.ErrExitCode{Code: code}
	}
	return nil
}

func serveCommand(args []string, e Environment) int {
	var roots rootFlags
	var bounds manlib.Limits
	var address, theme, assets string
	var system, remote bool
	f := flag.NewFlagSet("serve", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	f.Var(&roots, "root", "")
	f.IntVar(&bounds.MaxEntries, "max-pages", 10000, "")
	f.IntVar(&bounds.MaxTotalBytes, "max-total-bytes", 64<<20, "")
	f.IntVar(&bounds.MaxFileBytes, "max-file-bytes", 2<<20, "")
	f.StringVar(&address, "listen", "127.0.0.1:8080", "")
	f.BoolVar(&system, "system-search", false, "")
	f.BoolVar(&remote, "allow-remote", false, "")
	f.StringVar(&theme, "theme", "light", "")
	f.StringVar(&assets, "assets", "embed", "")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return writeString(e, serveHelp)
		}
		return usage(e, err.Error())
	}
	if f.NArg() != 0 {
		return usage(e, "serve requires 1..32 explicit --root directories and no positional files")
	}
	return runServe(address, roots, system, remote, bounds.MaxEntries, bounds.MaxTotalBytes, bounds.MaxFileBytes, theme, assets, e)
}

func runServe(address string, roots []string, system, remote bool, maxPages, maxTotalBytes, maxFileBytes int, theme, assets string, e Environment) int {
	if len(roots) < 1 || len(roots) > 32 {
		return usage(e, "serve requires 1..32 explicit --root directories and no positional files")
	}
	if maxPages < 1 || maxTotalBytes < 1 || maxFileBytes < 1 {
		return usage(e, "index limits must be positive")
	}
	bounds := manlib.Limits{
		MaxEntries:    maxPages,
		MaxTotalBytes: maxTotalBytes,
		MaxFileBytes:  maxFileBytes,
	}
	if err := manlib.ValidateLimits(bounds); err != nil {
		return usage(e, err.Error())
	}
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return usage(e, "listen requires HOST:PORT")
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 0 || n > 65535 {
		return usage(e, "listen port must be 0..65535")
	}
	addr, addrErr := netip.ParseAddr(host)
	if !remote && !strings.EqualFold(host, "localhost") && (addrErr != nil || !addr.IsLoopback()) {
		return usage(e, "non-loopback listen addresses require --allow-remote")
	}
	if strings.EqualFold(host, "localhost") {
		address = net.JoinHostPort("127.0.0.1", port)
	}
	if theme == "" {
		theme = "light"
	}
	if assets == "" {
		assets = "embed"
	}
	options := manlib.HandlerOptions{Render: renderhtml.Options{Theme: theme, Assets: renderhtml.AssetPolicy(assets)}}
	if err := renderhtml.ValidateOptions(options.Render); err != nil {
		return usage(e, err.Error())
	}
	if remote {
		options.AllowedHosts = []string{"*"}
	}
	if e.OpenRoot == nil || e.Listen == nil {
		return failure(e, "directory/listener capabilities are unavailable")
	}
	ctx := e.Context
	if ctx == nil {
		ctx = context.Background()
	}
	var opened []io.Closer
	var configured []manlib.Root
	closeRoots := func() error {
		var first error
		for _, c := range opened {
			if err := c.Close(); err != nil && first == nil {
				first = err
			}
		}
		opened = nil
		return first
	}
	defer func() { _ = closeRoots() }()
	for i, path := range roots {
		if err := ctx.Err(); err != nil {
			return ExitOK
		}
		root, owned, err := e.OpenRoot(path)
		if owned != nil {
			opened = append(opened, owned)
		}
		if err != nil {
			return failure(e, "open root: "+err.Error())
		}
		root.Label = fmt.Sprintf("root-%d", i+1)
		configured = append(configured, root)
	}
	index, err := manlib.Build(ctx, configured, bounds)
	closeErr := closeRoots()
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return ExitOK
		}
		return failure(e, "index library: "+err.Error())
	}
	if closeErr != nil {
		return failure(e, "close root: "+closeErr.Error())
	}
	if system {
		if e.SystemSearch == nil {
			options.Notices = append(options.Notices, "System search unavailable; built-in search remains active.")
		} else {
			provider, err := e.SystemSearch()
			if err != nil {
				options.Notices = append(options.Notices, "System search unavailable; built-in search remains active.")
			} else {
				options.SearchProvider = provider
			}
		}
	}
	handler, err := manlib.Handler(index, options)
	if err != nil {
		return failure(e, "HTTP handler: "+err.Error())
	}
	listener, err := e.Listen(address)
	if err != nil {
		return failure(e, "listen: "+err.Error())
	}
	defer listener.Close()
	if _, err := fmt.Fprintf(e.Err, "gtroff: serving %d pages at http://%s (read-only snapshot)\n", len(index.Entries()), listener.Addr()); err != nil {
		return ExitIO
	}
	if err := manlib.Serve(ctx, listener, handler); err != nil {
		return failure(e, "serve: "+err.Error())
	}
	return ExitOK
}
