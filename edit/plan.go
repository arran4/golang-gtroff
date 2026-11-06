package edit

import (
	"bytes"
	"context"
	"fmt"
	"sort"

	"github.com/arran4/golang-gtroff/query"
)

type budget struct {
	ctx  context.Context
	left int
}

func (m *budget) spend(n int) error {
	if err := m.ctx.Err(); err != nil {
		return err
	}
	if n < 0 || n > m.left {
		return ErrLimit
	}
	m.left -= n
	return nil
}

// Plan evaluates commands sequentially on a private candidate. Later selectors
// see earlier changes. Any failure leaves Working unchanged and returns no plan.
// Each command's patches use that step's source hashes/offsets, not initial ones.
func (w *Working) Plan(ctx context.Context, commands ...Command) (*Plan, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(commands) == 0 || len(commands) > w.o.MaxCommands {
		return nil, ErrOptions
	}
	m := &budget{ctx, w.o.MaxWork}
	candidate := *w
	candidate.files = cloneFiles(w.files)
	p := &Plan{owner: w.owner, version: w.version, before: cloneFiles(w.files)}
	for i, c := range commands {
		if err := m.spend(1); err != nil {
			return nil, err
		}
		if err := validateCommand(c); err != nil {
			return nil, fmt.Errorf("command %d: %w", i+1, err)
		}
		qo := query.Options{View: c.View, MaxWork: m.left, MaxResultBytes: 16 << 20, MaxMatches: w.o.MaxPatches}
		if m.left == 0 {
			return nil, ErrLimit
		}
		selected, err := query.Execute(ctx, candidate.set, c.Select, qo)
		if err != nil {
			return nil, err
		}
		if err = m.spend(selected.WorkUnits); err != nil {
			return nil, err
		}
		matches, err := restrict(selected.Matches, c.Restrict, w.o.Strict)
		if err != nil {
			return nil, err
		}
		if len(matches) == 0 {
			p.diagnostics = append(p.diagnostics, Diagnostic{Code: "no-matches", Message: "selector matched no targets"})
			continue
		}
		if len(matches) > 1 {
			p.diagnostics = append(p.diagnostics, Diagnostic{Code: "broadcast", Message: fmt.Sprintf("command %d selects %d logical matches", i+1, len(matches))})
		}
		patches, ds, err := candidate.commandPatches(ctx, c, matches, m)
		if err != nil {
			return nil, err
		}
		p.diagnostics = append(p.diagnostics, ds...)
		patches, ds, err = candidate.preparePatches(patches)
		if err != nil {
			return nil, err
		}
		p.diagnostics = append(p.diagnostics, ds...)
		if len(patches) > w.o.MaxPatches-len(p.patches) {
			return nil, ErrLimit
		}
		if err = candidate.applyPatches(ctx, patches, m); err != nil {
			return nil, fmt.Errorf("command %d: %w", i+1, err)
		}
		for j := range patches {
			patches[j].Step = i + 1
		}
		p.patches = append(p.patches, patches...)
	}
	p.after = cloneFiles(candidate.files)
	p.set = candidate.set
	return p, nil
}

// PlanPatches validates an explicit exact-byte patch batch. Every source must be
// captured and hashes must match. Included ownership and reparse policies are
// identical to selector commands. Caller-owned slices are copied immediately.
func (w *Working) PlanPatches(ctx context.Context, patches []Patch) (*Plan, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	m := &budget{ctx, w.o.MaxWork}
	if err := m.spend(1); err != nil {
		return nil, err
	}
	prepared, ds, err := w.preparePatches(patches)
	if err != nil {
		return nil, err
	}
	candidate := *w
	candidate.files = cloneFiles(w.files)
	if err = candidate.applyPatches(ctx, prepared, m); err != nil {
		return nil, err
	}
	for i := range prepared {
		prepared[i].Step = 1
	}
	return &Plan{owner: w.owner, version: w.version, before: cloneFiles(w.files), after: cloneFiles(candidate.files), set: candidate.set, patches: prepared, diagnostics: ds}, nil
}

func restrict(matches []query.Match, r Restriction, strict bool) ([]query.Match, error) {
	n := 0
	if r.Nth != 0 {
		n++
	}
	if r.From != 0 || r.To != 0 {
		n++
	}
	if r.FirstN != 0 {
		n++
	}
	if r.All {
		n++
	}
	if n > 1 || r.Nth < 0 || r.From < 0 || r.To < 0 || r.FirstN < 0 {
		return nil, ErrOptions
	}
	if r.Nth > 0 {
		if r.Nth > len(matches) {
			return nil, ErrSelection
		}
		matches = matches[r.Nth-1 : r.Nth]
	}
	if r.From != 0 || r.To != 0 {
		if r.From < 1 || r.To < r.From || r.To > len(matches) {
			return nil, ErrSelection
		}
		matches = matches[r.From-1 : r.To]
	}
	if r.FirstN > 0 && r.FirstN < len(matches) {
		matches = matches[:r.FirstN]
	}
	if strict && len(matches) != 1 {
		return nil, ErrSelection
	}
	return matches, nil
}

func validateCommand(c Command) error {
	if err := c.Select.Validate(); err != nil {
		return err
	}
	if c.View != "" && c.View != query.Syntax && c.View != query.Semantic {
		return ErrOptions
	}
	switch c.Operation {
	case ReplaceSource, Delete, InsertBefore, InsertAfter, AppendParagraph, AppendSection, AppendDefinition, ReplaceDescription:
	case FindReplaceSource:
		if c.Find == "" {
			return fmt.Errorf("%w: find must be nonempty", ErrOptions)
		}
	case MoveBefore, MoveAfter:
		if err := c.Destination.Validate(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%w: unknown operation %q", ErrOptions, c.Operation)
	}
	_, err := restrict(nil, c.Restrict, false)
	// Range validity against actual matches is checked after querying.
	if err == ErrSelection {
		return nil
	}
	return err
}

func (w *Working) preparePatches(patches []Patch) ([]Patch, []Diagnostic, error) {
	if len(patches) > w.o.MaxPatches {
		return nil, nil, ErrLimit
	}
	payloadBytes := 0
	for _, p := range patches {
		if len(p.Data) > w.o.MaxBytes-payloadBytes {
			return nil, nil, ErrLimit
		}
		payloadBytes += len(p.Data)
	}
	patches = clonePatches(patches)
	digests := map[string]string{}
	for _, p := range patches {
		data, ok := w.files[p.Source]
		if !ok {
			return nil, nil, ErrTarget
		}
		if p.Source != w.root && !w.o.IncludeEdits {
			return nil, nil, fmt.Errorf("%w: included file %q requires IncludeEdits", ErrTarget, p.Source)
		}
		if digests[p.Source] == "" {
			digests[p.Source] = digest(data)
		}
		if p.Digest == "" || p.Digest != digests[p.Source] {
			return nil, nil, ErrStale
		}
		if p.Start < 0 || p.End < p.Start || p.End > len(data) {
			return nil, nil, ErrTarget
		}
	}
	sort.SliceStable(patches, func(i, j int) bool {
		a, b := patches[i], patches[j]
		if a.Source != b.Source {
			return a.Source < b.Source
		}
		if a.Start != b.Start {
			return a.Start < b.Start
		}
		return a.End < b.End
	})
	var result []Patch
	var ds []Diagnostic
	type patchKey struct {
		source, owner, digest string
		start, end            int
	}
	seen := map[patchKey]bool{}
	for _, p := range patches {
		if bytes.Equal([]byte(w.files[p.Source][p.Start:p.End]), p.Data) {
			continue
		}
		// Include occurrences may interleave different selected nodes at one move
		// destination. Deduplicate by physical origin, never merely adjacent order.
		key := patchKey{p.Source, p.owner, digest(string(p.Data)), p.Start, p.End}
		if seen[key] {
			ds = append(ds, Diagnostic{Code: "repeated-source", Message: "identical physical patch selected more than once; applied once", Source: p.Source, Match: p.Match})
			continue
		}
		seen[key] = true
		result = append(result, p)
	}

	activeEnd, activeSource := -1, ""
	for _, p := range result {
		if p.Source != activeSource {
			activeSource = p.Source
			activeEnd = -1
		}
		if p.Start < activeEnd {
			return nil, nil, ErrOverlap
		}
		if p.End > p.Start {
			activeEnd = p.End
		}
	}
	return result, ds, nil
}

func (w *Working) applyPatches(ctx context.Context, patches []Patch, m *budget) error {
	next := cloneFiles(w.files)
	total := 0
	for _, s := range next {
		total += len(s)
	}
	for i := 0; i < len(patches); {
		j := i + 1
		for j < len(patches) && patches[j].Source == patches[i].Source {
			j++
		}
		s := w.files[patches[i].Source]
		size := len(s)
		for _, p := range patches[i:j] {
			if len(p.Data) > w.o.MaxBytes {
				return ErrLimit
			}
			size += len(p.Data) - (p.End - p.Start)
			if size < 0 || size > w.o.MaxBytes {
				return ErrLimit
			}
		}
		total += size - len(s)
		if total > w.o.MaxBytes {
			return ErrLimit
		}
		if err := m.spend(len(s) + size); err != nil {
			return err
		}
		var out bytes.Buffer
		out.Grow(size)
		at := 0
		for _, p := range patches[i:j] {
			out.WriteString(s[at:p.Start])
			out.Write(p.Data)
			at = p.End
		}
		out.WriteString(s[at:])
		next[patches[i].Source] = out.String()
		i = j
	}
	set, err := w.reparse(ctx, next, patches, m)
	if err != nil {
		return err
	}
	w.files = next
	w.set = set
	return nil
}
