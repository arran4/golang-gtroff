package script

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/arran4/golang-gtroff/edit"
	"github.com/arran4/golang-gtroff/scriptapi"
	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/parser"
)

// Cell, SessionOptions and Publisher share the engine-free session port.
type Cell = scriptapi.Cell
type SessionOptions = scriptapi.SessionOptions
type Publisher = scriptapi.Publisher

// ErrPending prevents another cell from overwriting unpublished output/intents.
var ErrPending = scriptapi.ErrPending

// Session retains native Tengo globals, functions and one current document.
// It has no I/O capabilities. Methods require exclusive access and publishers
// must not reenter. A successful cell is pending until explicitly published.
type Session struct {
	ctx                                context.Context
	o                                  SessionOptions
	r                                  *runtime
	files                              *parser.SourceFileSet
	globals, constants                 []tengo.Object
	bindings                           []string
	pending                            *Result
	cells, bytes, steps, calls, output int
	probeBytes                         int
}

func sessionDefaults(o SessionOptions) (SessionOptions, error) {
	var err error
	o.Cell, err = defaults(o.Cell)
	if err != nil {
		return o, err
	}
	if o.MaxCells <= 0 {
		o.MaxCells = 1000
	}
	if o.MaxSessionBytes <= 0 {
		o.MaxSessionBytes = 4 << 20
	}
	if o.MaxSessionSteps <= 0 {
		o.MaxSessionSteps = 1000000
	}
	if o.MaxSessionHostCalls <= 0 {
		o.MaxSessionHostCalls = 10000
	}
	if o.MaxSessionOutputBytes <= 0 {
		o.MaxSessionOutputBytes = 32 << 20
	}
	if o.MaxStateObjects <= 0 {
		o.MaxStateObjects = 100000
	}
	if o.MaxStateBytes <= 0 {
		o.MaxStateBytes = 16 << 20
	}
	if o.MaxCells > 10000 || o.MaxSessionBytes > 64<<20 || o.MaxSessionSteps > 100000000 || o.MaxSessionHostCalls > 1000000 || o.MaxSessionOutputBytes > 128<<20 || o.MaxStateObjects > 1000000 || o.MaxStateBytes > 64<<20 || o.Cell.MaxConstants > 65535 {
		return o, ErrOptions
	}
	return o, nil
}

// NewSession creates persistent state from already captured sources. Code must
// be empty; use Eval for each cell. Original guards survive every cell and fork;
// only an explicit successful Commit in Publish advances the write baseline.
func NewSession(ctx context.Context, request Request, o SessionOptions) (*Session, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if request.InputError != nil {
		return nil, request.InputError
	}
	if len(request.Code) != 0 {
		return nil, fmt.Errorf("%w: NewSession accepts no program; use Eval", ErrOptions)
	}
	var err error
	o, err = sessionDefaults(o)
	if err != nil {
		return nil, err
	}
	s := &Session{ctx: ctx, o: o, r: &runtime{o: o.Cell, ctx: ctx}, files: parser.NewFileSet(), globals: make([]tengo.Object, tengo.GlobalsSize), bindings: []string{"doc", checkpointName}}
	if request.Sources != nil {
		s.r.working, err = edit.New(request.Sources, o.Cell.Edit)
		if err != nil {
			return nil, err
		}
	}
	doc := s.r.documentObject().(*tengo.ImmutableMap)
	checkpoint := &tengo.UserFunction{Name: checkpointName, Value: s.r.checkpoint}
	s.globals[0] = doc
	s.globals[1] = checkpoint
	return s, nil
}

// Eval compiles and executes one native Tengo cell. Candidate globals (including
// cycles, shared slice/map backing and closure cells) and the working document
// are isolated before execution. Errors publish none of that candidate state,
// output or write requests, but consume cumulative attempted-work budgets.
func (s *Session) Eval(ctx context.Context, cell Cell) (result *Result, err error) {
	result = &Result{}
	if s.pending != nil {
		return result, ErrPending
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err = s.ctx.Err(); err != nil {
		return result, err
	}
	if err = ctx.Err(); err != nil {
		return result, err
	}
	if s.cells >= s.o.MaxCells {
		return result, ErrLimit
	}
	s.cells++
	if len(cell.Code) > s.o.MaxSessionBytes-s.bytes {
		s.bytes = s.o.MaxSessionBytes
		return result, ErrLimit
	}
	s.bytes += len(cell.Code)
	if len(cell.Code) > s.o.Cell.MaxScriptBytes {
		return result, ErrLimit
	}
	o := s.o.Cell
	o.MaxSteps = min(o.MaxSteps, s.o.MaxSessionSteps-s.steps)
	o.MaxHostCalls = min(o.MaxHostCalls, s.o.MaxSessionHostCalls-s.calls)
	o.MaxOutputBytes = min(o.MaxOutputBytes, s.o.MaxSessionOutputBytes-s.output)
	if o.MaxSteps <= 0 || o.MaxHostCalls <= 0 || o.MaxOutputBytes <= 0 {
		return result, ErrLimit
	}
	// Keep synchronous deadline checks useful on js/wasm as well as cancellation.
	deadline := time.Now().Add(o.Duration)
	if d, ok := s.ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	ctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	stop := context.AfterFunc(s.ctx, cancel)
	defer stop()
	previous := *s.r
	candidate := runtime{ctx: ctx, o: o, created: previous.created}
	if previous.working != nil {
		candidate.working = previous.working.Fork()
	}
	*s.r = candidate
	defer func() {
		result.Stats.HostCalls = s.r.calls
		result.Stats.Steps = s.r.steps
		result.Stats.OutputBytes = len(s.r.output)
		s.steps += s.r.steps
		s.calls += s.r.calls
		usedOutput := len(s.r.output)
		for _, write := range s.r.writes {
			usedOutput += len(write.Data)
		}
		s.output += usedOutput
		if err != nil {
			*s.r = previous
			err = normalizeLimit(err)
		}
	}()
	joined := append(append([]tengo.Object(nil), s.globals...), s.constants...)
	state, err := cloneState(ctx, joined, s.o)
	if err != nil {
		return result, err
	}
	globals, constants := state[:tengo.GlobalsSize], state[tengo.GlobalsSize:]
	if cell.Name == "" {
		cell.Name = fmt.Sprintf("<cell-%d>", s.cells)
	}
	file := s.files.AddFile(cell.Name, -1, len(cell.Code))
	started := time.Now()
	if err = preflightScript(ctx, file, cell.Code, o); err != nil {
		return result, err
	}
	parsed, err := parser.NewParser(file, cell.Code, nil).ParseFile()
	if err != nil {
		return result, err
	}
	if err = nativeShape(ctx, parsed, 4*o.MaxTokens); err != nil {
		return result, err
	}
	symbols := tengo.NewSymbolTable()
	compiler := tengo.NewCompiler(file, symbols, constants, tengo.NewModuleMap(), nil)
	compiler.EnableFileImport(false)
	for i, name := range s.bindings {
		if name == "" {
			name = fmt.Sprintf("__gtroff_slot_%d", i)
		}
		symbols.Define(name)
	}
	if err = instrument(ctx, parsed, 2*o.MaxTokens); err != nil {
		return result, err
	}
	if err = compiler.Compile(parsed); err != nil {
		return result, err
	}
	code := compiler.Bytecode()
	// Do not deduplicate/reindex constants: retained functions reference previous
	// cells' absolute indices. Compiler and VM operands are bounded before Run.
	if err = nativeConstantBounds(code); err != nil {
		return result, err
	}
	result.Stats.Constants = len(code.Constants)
	if len(code.Constants) > o.MaxConstants || symbols.MaxSymbols() > tengo.GlobalsSize {
		return result, ErrLimit
	}
	result.Stats.CompileDuration = time.Since(started)
	if err = ctx.Err(); err != nil {
		return result, err
	}
	started = time.Now()
	err = runVM(ctx, tengo.NewVM(code, globals, o.MaxAllocs))
	result.Stats.RunDuration = time.Since(started)
	if s.r.hostError != nil && !errors.Is(err, s.r.hostError) {
		err = errors.Join(err, s.r.hostError)
	}
	if err != nil {
		return result, err
	}
	if len(s.r.writes) > 0 && (s.r.working != s.r.writeWorking || s.r.working.Version() != s.r.writeVersion) {
		return result, edit.ErrStale
	}
	usedOutput := len(s.r.output)
	for _, write := range s.r.writes {
		usedOutput += len(write.Data)
	}
	if usedOutput > s.o.MaxSessionOutputBytes-s.output {
		return result, ErrLimit
	}

	// Compiler-private global iterator slots are unreachable from Tengo source;
	// each loop overwrites its slot before use. Do not retain iterator internals.
	for i, v := range globals {
		if _, ok := v.(tengo.Iterator); ok {
			globals[i] = nil
		}
	}
	state, err = cloneState(ctx, append(globals, code.Constants...), s.o)
	if err != nil {
		return result, err
	}
	bindings := make([]string, symbols.MaxSymbols())
	for _, name := range symbols.Names() {
		sym, _, ok := symbols.Resolve(name, false)
		if ok && sym.Scope == tengo.ScopeGlobal {
			bindings[sym.Index] = name
		}
	}
	if err = ctx.Err(); err != nil {
		return result, err
	}
	s.globals = state[:tengo.GlobalsSize]
	s.constants = state[tengo.GlobalsSize:]
	s.bindings = bindings
	result.Working = s.r.working
	result.Created = s.r.created
	result.Output = s.r.output
	result.Writes = s.r.writes
	result.Diagnostics = s.r.diagnostics
	if result.Working != nil {
		result.SourceDiagnostics = result.Working.Sources().Diagnostics()
	}
	result.Stats.HostCalls = s.r.calls
	result.Stats.Steps = s.r.steps
	result.Stats.OutputBytes = len(s.r.output)
	s.pending = copyResult(result)
	return copyResult(result), nil
}

// Pending returns an isolated snapshot of the last successful unpublished cell,
// or nil if none exists. Mutating a returned Working cannot alter the session.
func (s *Session) Pending(ctx context.Context) (*Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return copyResult(s.pending), nil
}

// Publish invokes an explicit host capability for the pending result. Failure
// retains it unchanged for inspection/retry and prevents a subsequent Eval.
// Success acknowledges emissions/intents and adopts any successful working
// Commit performed by the publisher. Use a no-op publisher to discard output.
func (s *Session) Publish(ctx context.Context, publish Publisher) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.pending == nil {
		return nil
	}
	if publish == nil {
		return fmt.Errorf("%w: explicit publisher required", ErrOptions)
	}
	result := copyResult(s.pending)
	if err := publish(ctx, result); err != nil {
		return err
	}
	s.r.working = result.Working
	s.r.created = result.Created
	s.pending = nil
	return nil
}
func copyResult(r *Result) *Result {
	if r == nil {
		return nil
	}
	out := *r
	if r.Working != nil {
		out.Working = r.Working.Fork()
	}
	out.Output = append([]byte(nil), r.Output...)
	out.Diagnostics = append(out.Diagnostics[:0:0], r.Diagnostics...)
	out.SourceDiagnostics = append(out.SourceDiagnostics[:0:0], r.SourceDiagnostics...)
	out.Writes = append(out.Writes[:0:0], r.Writes...)
	for i := range out.Writes {
		out.Writes[i].Data = append([]byte(nil), out.Writes[i].Data...)
	}
	return &out
}
