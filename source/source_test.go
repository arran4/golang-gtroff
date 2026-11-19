package source_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/source"
)

func TestSourceGraphAndPhysicalOwnership(t *testing.T) {
	fsys := fstest.MapFS{
		"main.1":          {Data: []byte(".SH ENVIRONMENT\n.so sub/common.roff\n.so sub/common.roff\nend\n")},
		"sub/common.roff": {Data: []byte(".TP\n.B TOOL_HOME\nDirectory\n.mso ../tail.roff\n")},
		"tail.roff":       {Data: []byte(".\\\" comment\r\n雪\r\n")},
	}
	set, err := source.Load(context.Background(), fsys, "main.1", source.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Files()) != 5 || len(set.Edges()) != 4 || len(set.Diagnostics()) != 0 {
		t.Fatalf("graph cardinality: files %d edges %+v diagnostics %+v", len(set.Files()), set.Edges(), set.Diagnostics())
	}
	if set.Files()[1].ID() == set.Files()[3].ID() || set.Files()[1].Name() != set.Files()[3].Name() {
		t.Fatal("repeated include identity lost")
	}
	for _, edge := range set.Edges() {
		parent := set.File(edge.From)
		child := set.File(edge.To)
		if edge.State != "resolved" || child == nil || child.ParentEdge() != edge.ID || child.Depth() != parent.Depth()+1 {
			t.Fatalf("broken provenance: %+v", edge)
		}
		node := parent.AST().Node(edge.Node)
		if node == nil || node.Span() != edge.Span || node.Attributes().Name != edge.Request {
			t.Fatal("directive identity changed")
		}
	}
	collected := map[source.ID]*strings.Builder{}
	for _, line := range set.Lines() {
		file := set.File(line.Source)
		n := set.Node(line)
		if n == nil || file.AST().Node(line.Node) != n || n.Kind() != groff.ASTLine || line.Depth != file.Depth() {
			t.Fatal("logical sequence invented syntax")
		}
		if collected[line.Source] == nil {
			collected[line.Source] = new(strings.Builder)
		}
		collected[line.Source].WriteString(n.Raw())
	}
	for _, f := range set.Files() {
		if collected[f.ID()].String() != string(fsys[f.Name()].Data) || !f.AST().SourceComplete() {
			t.Fatal("physical source changed")
		}
	}
	first := set.Lines()[2]
	if set.File(first.Source).Name() != "sub/common.roff" {
		t.Fatal("include child not in logical source order")
	}
	before, err := json.Marshal(set)
	if err != nil {
		t.Fatal(err)
	}
	other, err := source.Load(context.Background(), fsys, "main.1", source.Options{})
	if err != nil {
		t.Fatal(err)
	}
	after, err := json.Marshal(other)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("nondeterministic graph", err)
	}
	lines := set.Lines()
	lines[0].Source = 0
	edges := set.Edges()
	edges[0].State = "changed"
	files := set.Files()
	files[0] = nil
	if set.Lines()[0].Source == 0 || set.Edges()[0].State != "resolved" || set.Files()[0] == nil {
		t.Fatal("slice accessor aliases state")
	}
	if set.File(0) != nil || set.File(^source.ID(0)) != nil || set.Node(source.Line{}) != nil {
		t.Fatal("invalid source identity accepted")
	}
}

func TestIncludeFailuresAndNonexecution(t *testing.T) {
	cases := []struct {
		name, input, state string
		fsys               fstest.MapFS
		noIncludes         bool
	}{
		{name: "missing", input: ".so missing.roff\n", state: "missing", fsys: fstest.MapFS{}},
		{name: "escape", input: ".so ../outside\n", state: "denied", fsys: fstest.MapFS{}},
		{name: "absolute", input: ".so /outside\n", state: "denied", fsys: fstest.MapFS{}},
		{name: "windows", input: ".so C:/outside\n", state: "invalid", fsys: fstest.MapFS{}},
		{name: "indirect", input: ".so \\*[NAME]\n", state: "invalid", fsys: fstest.MapFS{}},
		{name: "quoted", input: ".so \"broken\n", state: "invalid", fsys: fstest.MapFS{}},
		{name: "extra", input: ".so a b\n", state: "invalid", fsys: fstest.MapFS{}},
		{name: "no context", input: ".so child\n", state: "unresolved"},
		{name: "disabled", input: ".so child\n", state: "disabled", fsys: fstest.MapFS{"child": {Data: []byte("loaded\n")}}, noIncludes: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o := source.Options{NoIncludes: tc.noIncludes}
			if tc.fsys != nil {
				o.FS = tc.fsys
			}
			set, err := source.Read(context.Background(), strings.NewReader(tc.input), "main.1", o)
			if err != nil || len(set.Files()) != 1 || len(set.Edges()) != 1 || set.Edges()[0].State != tc.state {
				t.Fatalf("unexpected failure state: %+v, %v", set.Edges(), err)
			}
			if set.File(1).AST().Source() != tc.input || len(set.Diagnostics()) == 0 {
				t.Fatal("failed directive disappeared")
			}
			o.Strict = true
			set, err = source.Read(context.Background(), strings.NewReader(tc.input), "main.1", o)
			if !errors.Is(err, source.ErrStrict) || set == nil {
				t.Fatal("strict failure lost evidence", err)
			}
			var include *source.IncludeError
			if !errors.As(err, &include) || include.Edge.State != tc.state {
				t.Fatal("strict failure lost classification", err)
			}
			if tc.name == "missing" && !errors.Is(err, fs.ErrNotExist) {
				t.Fatal("missing error identity lost")
			}
		})
	}
	input := ".de SAVED\n.so secret\n..\n.TS\n.mso secret\n.TE\n.if 0 .so secret\n"
	set, err := source.Read(context.Background(), strings.NewReader(input), "main.1", source.Options{FS: panicFS{}})
	if err != nil || len(set.Edges()) != 0 || set.File(1).AST().Source() != input {
		t.Fatal("opaque/runtime includes interpreted", err)
	}
}

type panicFS struct{}

func (panicFS) Open(string) (fs.File, error) { panic("filesystem unexpectedly consulted") }

func TestCyclesLimitsAndCancellation(t *testing.T) {
	fsys := fstest.MapFS{"a": {Data: []byte(".so b\n")}, "b": {Data: []byte(".so a\n")}}
	set, err := source.Load(context.Background(), fsys, "a", source.Options{})
	if err != nil || len(set.Files()) != 2 || set.Edges()[1].State != "cycle" {
		t.Fatal("cycle not preserved", err)
	}
	for _, o := range []source.Options{{MaxDepth: 1}, {MaxFiles: 1}, {MaxBytes: 8}, {MaxNodes: 8}} {
		// A non-cyclic chain distinguishes a resource bound from cycle detection.
		chain := fstest.MapFS{"a": {Data: []byte(".so b\n")}, "b": {Data: []byte(".so c\n")}, "c": {Data: []byte("tail\n")}}
		set, err := source.Load(context.Background(), chain, "a", o)
		if err == nil || set == nil {
			t.Fatalf("limit ignored: %+v", o)
		}
		if o.MaxNodes != 0 {
			total := 0
			for _, f := range set.Files() {
				total += len(f.AST().Root().Descendants()) + 1
			}
			if total > o.MaxNodes+1 {
				t.Fatal("node budget restarted for child")
			}
		}
		if o.MaxBytes != 0 {
			total := 0
			for _, f := range set.Files() {
				total += len(f.AST().Source())
			}
			if total > o.MaxBytes+1 {
				t.Fatal("byte budget restarted for child")
			}
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	set, err = source.Load(ctx, panicFS{}, "a", source.Options{})
	if !errors.Is(err, context.Canceled) || len(set.Files()) != 0 {
		t.Fatal("cancelled load opened a source", err)
	}
	ctx, cancel = context.WithCancel(context.Background())
	r := &cancelReader{cancel: cancel}
	set, err = source.Read(ctx, r, "stdin", source.Options{FS: panicFS{}})
	if !errors.Is(err, context.Canceled) || set.File(1).AST().Source() != ".so child\n" || set.Edges()[0].State != "incomplete" {
		t.Fatal("cancelled read lost prefix or opened child", err)
	}
}

type cancelReader struct{ cancel context.CancelFunc }

func (r *cancelReader) Read(p []byte) (int, error) { r.cancel(); return copy(p, ".so child\n"), nil }

func TestOSRootSymlinksAndAliases(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret"), []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.1"), []byte(".so escape\n.so alias\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "secret"), filepath.Join(dir, "escape")); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	if err := os.Symlink("main.1", filepath.Join(dir, "alias")); err != nil {
		t.Fatal(err)
	}
	root, err := source.OpenDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	set, err := source.Load(context.Background(), root.FS(), "main.1", source.Options{})
	if err != nil || len(set.Files()) != 1 || set.Edges()[0].State != "denied" || set.Edges()[1].State != "cycle" {
		t.Fatalf("escape/alias not bounded: %+v %v", set.Edges(), err)
	}
}

var readerFailure = errors.New("source read failed")

type failedReader struct{}

func (failedReader) Read(p []byte) (int, error) { return copy(p, "prefix"), readerFailure }
func TestReaderFailuresAndExactBytes(t *testing.T) {
	for _, input := range []string{"text\r\n雪\r\n", "invalid \xff\n.so missing\n"} {
		set, err := source.Read(context.Background(), strings.NewReader(input), "stdin", source.Options{})
		if err != nil || set.File(1).AST().Source() != input {
			t.Fatal("source changed", err)
		}
	}
	set, err := source.Read(context.Background(), failedReader{}, "stdin", source.Options{})
	if !errors.Is(err, readerFailure) || set.File(1).AST().Source() != "prefix" || set.File(1).AST().SourceComplete() {
		t.Fatal("reader prefix/error lost", err)
	}
	var nilReader *bytes.Reader
	if _, err := source.Read(context.Background(), nilReader, "stdin", source.Options{}); err == nil {
		t.Fatal("typed nil reader accepted")
	}
	if _, err := source.Load(context.Background(), nil, "main.1", source.Options{}); err == nil {
		t.Fatal("missing root FS accepted")
	}
	if _, err := source.Load(context.Background(), panicFS{}, "../escape", source.Options{}); !errors.Is(err, fs.ErrInvalid) {
		t.Fatal("invalid root opened", err)
	}
	// A source-exact parser still has no implicit include behavior.
	tree, err := groff.ParseAST(strings.NewReader(".so missing\n"), groff.WithIncludeResolver(groff.NewSafeDirectoryIncludeResolver("/unavailable", 1)))
	if err != nil || tree.Source() != ".so missing\n" {
		t.Fatal("low-level AST changed include contract", err)
	}
}

var _ io.Reader = failedReader{}

type trackingFS struct {
	fs.FS
	opened, closed int
	closeErr       error
}

func (f *trackingFS) Open(name string) (fs.File, error) {
	file, err := f.FS.Open(name)
	if err != nil {
		return nil, err
	}
	f.opened++
	return &trackingFile{File: file, owner: f}, nil
}

type trackingFile struct {
	fs.File
	owner *trackingFS
}

func (f *trackingFile) Close() error {
	f.owner.closed++
	return errors.Join(f.File.Close(), f.owner.closeErr)
}

func TestOwnedFilesAndFailureBudgets(t *testing.T) {
	for _, closeErr := range []error{nil, readerFailure} {
		fsys := &trackingFS{FS: fstest.MapFS{"main": {Data: []byte(".so child\n")}, "child": {Data: []byte("text\n")}}, closeErr: closeErr}
		set, err := source.Load(context.Background(), fsys, "main", source.Options{})
		if fsys.opened != fsys.closed {
			t.Fatal("owned file leaked")
		}
		if closeErr != nil {
			if !errors.Is(err, closeErr) || len(set.Files()) != 1 || set.Edges()[0].State != "incomplete" {
				t.Fatal("close failure did not stop child access", err)
			}
		} else if err != nil || fsys.opened != 2 {
			t.Fatal("unexpected successful ownership", err)
		}
	}
	root := ".so child\n.so child\n"
	fsys := fstest.MapFS{"main": {Data: []byte(root)}, "child": {Data: []byte("long content\n")}}
	set, err := source.Load(context.Background(), fsys, "main", source.Options{MaxBytes: len(root) + 2})
	if !errors.Is(err, source.ErrLimit) || len(set.Files()) != 2 || set.Edges()[0].State != "incomplete" || set.Edges()[1].State != "incomplete" {
		t.Fatal("failed child restarted byte budget", err)
	}
	total := 0
	for _, file := range set.Files() {
		total += len(file.AST().Source())
	}
	if total != len(root)+3 {
		t.Fatal("detection byte not accounted for", total)
	}
}

type explicitHandler struct{}

func (explicitHandler) CanHandle(name, pkg string) bool { return name == ".CUSTOM" }
func (explicitHandler) Parse(name, args, pkg string) (groff.Token, error) {
	return groff.TokenText("explicit"), nil
}
func TestExplicitRegistry(t *testing.T) {
	registry := groff.NewRegistry()
	registry.RegisterCustom(explicitHandler{})
	set, err := source.Read(context.Background(), strings.NewReader(".CUSTOM\n"), "stdin", source.Options{Registry: registry})
	if err != nil || len(set.Diagnostics()) != 0 {
		t.Fatal("explicit registry ignored", err)
	}
	set, err = source.Read(context.Background(), strings.NewReader(".CUSTOM\n"), "stdin", source.Options{})
	if err != nil || len(set.Diagnostics()) != 1 {
		t.Fatal("default load acquired custom registry", err)
	}
}
