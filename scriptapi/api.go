// Package scriptapi defines the engine-free port used by the core CLI and the
// optional Tengo module. It carries captured documents and staged results only;
// it neither executes scripts nor acquires I/O capabilities.
package scriptapi

import (
	"context"
	"errors"
	"time"

	"github.com/arran4/golang-gtroff/edit"
	"github.com/arran4/golang-gtroff/source"
)

var (
	ErrOptions  = errors.New("invalid script options")
	ErrLimit    = errors.New("script resource limit exceeded")
	ErrDenied   = errors.New("script capability denied")
	ErrDocument = errors.New("script has no current document")
)

// Request supplies an explicit script and already captured document. InputError
// must be handled before execution. With no Sources, scripts may create an
// in-memory document only when AllowEdits is explicit.
type Request struct {
	Name       string
	Code       []byte
	Sources    *source.Set
	InputError error
}

// Options grants only in-memory mutation and declarative write-request creation.
// The engine itself never writes files. Work limits apply to individual bounded
// host operations; MaxHostCalls and Duration bound the enclosing execution.
// MaxSteps counts inserted statement/block checkpoints, not VM instructions.
// Tengo's object allocation count is not a byte quota; global Tengo value limits
// remain owned by the embedding process and are never changed here.
type Options struct {
	AllowEdits, AllowWriteRequests                                                    bool
	Edit                                                                              edit.Options
	MaxScriptBytes, MaxTokens, MaxNesting, MaxConstants, MaxHostCalls, MaxOutputBytes int
	MaxSteps                                                                          int
	MaxAllocs                                                                         int64
	Duration                                                                          time.Duration
}

// WriteRequest is a reviewable intent, not an executed filesystem operation.
// Data is copied from the working source, never trusted from a script record.
// A host must authorize Destination and enforce original guards before writing.
type WriteRequest struct {
	Source, Destination string
	Data                []byte
}

type Stats struct {
	HostCalls, OutputBytes, Constants int
	Steps                             int
	CompileDuration, RunDuration      time.Duration
}

// Result exposes working changes/output only after successful execution. Error
// results retain statistics but never publish staged emissions or partial edits.
// Working remains mutable and its eventual explicit write policy is host-owned.
type Result struct {
	// Created marks a new virtual document, requiring a copy destination.
	Created           bool
	Working           *edit.Working
	Output            []byte
	Writes            []WriteRequest
	Diagnostics       []edit.Diagnostic
	SourceDiagnostics []source.Diagnostic
	Stats             Stats
}

// Backend executes one explicit request with staged, non-I/O results.
type Backend func(context.Context, Request, Options) (*Result, error)
