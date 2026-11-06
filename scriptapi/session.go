package scriptapi

import (
	"context"
	"errors"
)

// ErrPending identifies a successful cell awaiting explicit host publication.
var ErrPending = errors.New("script session has an unpublished successful cell")

// Cell is one native Tengo program in a persistent session. Name labels errors;
// no host file is opened. Queries stored in globals remain snapshot data.
type Cell struct {
	Name string
	Code []byte
}

// SessionOptions bounds individual cells, cumulative attempted work and the
// retained VM value graph. Failed cells also consume session budgets. State
// accounting is conservative and is not a process heap-byte quota.
type SessionOptions struct {
	Cell                                                                                   Options
	MaxCells, MaxSessionBytes, MaxSessionSteps, MaxSessionHostCalls, MaxSessionOutputBytes int
	MaxStateObjects, MaxStateBytes                                                         int
}

// Publisher performs explicit host output for a successful cell. Result.Working
// is an isolated fork; a successful Commit on that fork advances the session's
// baseline only if Publisher succeeds. Failure keeps the original pending result
// for inspection/retry. A publisher may already have written a stream prefix;
// the host must report its own filesystem recovery guarantees accurately.
type Publisher func(context.Context, *Result) error

// Session retains native variables/functions and original document baselines.
// Eval is transactional. Successful Eval must be followed by Publish before the
// next cell; use a no-op publisher to explicitly accept/discard staged output.
// Pending returns isolated copies. Methods require exclusive caller access;
// publishers must not reenter the session. No method implicitly opens files.
type Session interface {
	// Ready probes native syntax without executing or defining symbols.
	Ready(context.Context, Cell) (bool, error)
	Eval(context.Context, Cell) (*Result, error)
	Pending(context.Context) (*Result, error)
	Publish(context.Context, Publisher) error
}

// SessionFactory creates an optional native engine session. Request.Code must be
// empty: use Eval for cells. InputError and captured Sources retain their usual
// loading contracts; no sources are reopened by the engine.
type SessionFactory func(context.Context, Request, SessionOptions) (Session, error)
