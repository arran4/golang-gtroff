package docdiff

import (
	"sort"
	"strconv"
)

type pair struct {
	before, after *entity
	moved         bool
	reason        string
}
type matcher struct {
	report      *Report
	options     Options
	meter       *meter
	pairs       []*pair
	resultBytes int
}

func (m *matcher) compare(before, after []*entity) error {
	if err := m.matchGroups(before, after, false); err != nil {
		return err
	}
	if err := m.matchGroups(before, after, true); err != nil {
		return err
	}
	// Longest increasing subsequences distinguish actual relative reorder from
	// ordinal shifts caused by insertions/removals, in O(n log n) bounded work.
	groups := map[string][]*pair{}
	for _, p := range m.pairs {
		if err := m.meter.spend(1); err != nil {
			return err
		}
		if p.before.object.Scope != p.after.object.Scope {
			p.moved = true
			p.reason = "unique identity changed structural/source scope"
			continue
		}
		kind := p.before.object.Kind
		if kind == Metadata || kind == Preamble || kind == SourceFile {
			continue
		}
		key := tuple(string(kind), p.before.object.Scope)
		groups[key] = append(groups[key], p)
	}
	for _, key := range sortedKeys(groups) {
		group := groups[key]
		sort.Slice(group, func(i, j int) bool { return group[i].before.position < group[j].before.position })
		tails := []int{}
		tailIndices := []int{}
		previous := make([]int, len(group))
		for index, p := range group {
			if err := m.meter.spend(1); err != nil {
				return err
			}
			at := sort.SearchInts(tails, p.after.position)
			previous[index] = -1
			if at > 0 {
				previous[index] = tailIndices[at-1]
			}
			if at == len(tails) {
				tails = append(tails, p.after.position)
				tailIndices = append(tailIndices, index)
			} else {
				tails[at] = p.after.position
				tailIndices[at] = index
			}
		}
		kept := map[int]bool{}
		if len(tailIndices) > 0 {
			for at := tailIndices[len(tailIndices)-1]; at >= 0; at = previous[at] {
				kept[at] = true
			}
		}
		for index, p := range group {
			if !kept[index] {
				p.moved = true
				p.reason = "relative order changed among matched siblings"
			}
		}
	}
	for _, p := range m.pairs {
		changed := p.before.object.Digest != p.after.object.Digest
		if !changed && !p.moved {
			continue
		}
		kind := Changed
		if !changed {
			kind = Moved
		}
		reason := p.reason
		if reason == "" {
			reason = "compared content payload changed"
		}
		if err := m.emit(Change{Kind: kind, Before: copyObject(p.before.object), After: copyObject(p.after.object), Moved: p.moved, ContentChanged: changed, Confidence: confidence(p.before.object, p.after.object), Reason: reason}); err != nil {
			return err
		}
	}
	for _, e := range before {
		if !e.paired {
			if err := m.emit(Change{Kind: Removed, Before: copyObject(e.object), Confidence: confidence(e.object), Reason: "identity has no counterpart in the inspected after snapshot"}); err != nil {
				return err
			}
		}
	}
	for _, e := range after {
		if !e.paired {
			if err := m.emit(Change{Kind: Added, After: copyObject(e.object), Confidence: confidence(e.object), Reason: "identity has no counterpart in the inspected before snapshot"}); err != nil {
				return err
			}
		}
	}
	return nil
}

func sortChanges(report *Report) {
	sort.SliceStable(report.Changes, func(i, j int) bool {
		a, b := changeObject(report.Changes[i]), changeObject(report.Changes[j])
		if a.Location.SourceKey != b.Location.SourceKey {
			return a.Location.SourceKey < b.Location.SourceKey
		}
		if a.Location.Span.Start.Offset != b.Location.Span.Start.Offset {
			return a.Location.Span.Start.Offset < b.Location.Span.Start.Offset
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Identity != b.Identity {
			return a.Identity < b.Identity
		}
		return report.Changes[i].Kind < report.Changes[j].Kind
	})
	for i := range report.Changes {
		report.Changes[i].Index = i + 1
	}
}
func (m *matcher) matchGroups(before, after []*entity, relocated bool) error {
	old, new := map[string][]*entity{}, map[string][]*entity{}
	eligible := func(e *entity) bool {
		return !e.paired && (!relocated || e.object.Kind != Metadata && e.object.Kind != Preamble && e.object.Kind != SourceFile)
	}
	keyFor := func(e *entity) string {
		if relocated {
			return tuple(string(e.object.Kind), e.object.Identity)
		}
		return tuple(string(e.object.Kind), e.object.Scope, e.object.Identity)
	}
	for _, entry := range []struct {
		items  []*entity
		groups map[string][]*entity
	}{{before, old}, {after, new}} {
		for _, e := range entry.items {
			if err := m.meter.spend(len(e.object.Identity) + len(e.object.Scope) + 1); err != nil {
				return err
			}
			if eligible(e) {
				key := keyFor(e)
				entry.groups[key] = append(entry.groups[key], e)
			}
		}
	}
	for _, key := range sortedKeys(old) {
		a, b := old[key], new[key]
		if len(b) == 0 {
			continue
		}
		if len(a) == 1 && len(b) == 1 {
			a[0].paired = true
			b[0].paired = true
			m.pairs = append(m.pairs, &pair{before: a[0], after: b[0]})
			continue
		}
		// Identical original snapshots or unique content payloads can resolve
		// repeated labels without relying on volatile result ordinals.
		if err := m.alignUnique(a, b, snapshotIdentity); err != nil {
			return err
		}
		a, b = unpaired(a), unpaired(b)
		if err := m.alignUnique(a, b, func(e *entity) string { return e.object.Digest }); err != nil {
			return err
		}
		a, b = unpaired(a), unpaired(b)
		if len(a) == 0 || len(b) == 0 {
			continue
		}
		if len(a) == 1 && len(b) == 1 {
			a[0].paired = true
			b[0].paired = true
			m.pairs = append(m.pairs, &pair{before: a[0], after: b[0]})
			continue
		}
		ambiguity := Ambiguity{SchemaVersion: 1, Kind: a[0].object.Kind, Identity: a[0].object.Identity, Before: []Object{}, After: []Object{}, Reason: "multiple candidates share an identity; no ordinal-based pairing is asserted"}
		size := 256
		for _, group := range [][]*entity{a, b} {
			for _, e := range group {
				if err := m.meter.spend(len(e.object.Value) + 1); err != nil {
					return err
				}
				size += objectSize(e.object)
				e.paired = true
			}
		}
		if len(m.report.Ambiguities)+len(m.report.Changes) >= m.options.MaxChanges || size > m.options.MaxResultBytes-m.resultBytes {
			return ErrLimit
		}
		m.resultBytes += size
		for _, e := range a {
			ambiguity.Before = append(ambiguity.Before, e.object)
		}
		for _, e := range b {
			ambiguity.After = append(ambiguity.After, e.object)
		}
		m.report.Ambiguities = append(m.report.Ambiguities, ambiguity)
	}
	return nil
}
func (m *matcher) emit(change Change) error {
	if len(m.report.Changes)+len(m.report.Ambiguities) >= m.options.MaxChanges {
		return ErrLimit
	}
	size := 256 + len(change.Reason)
	if change.Before != nil {
		size += objectSize(*change.Before)
	}
	if change.After != nil {
		size += objectSize(*change.After)
	}
	if size > m.options.MaxResultBytes-m.resultBytes {
		return ErrLimit
	}
	if err := m.meter.spend(size); err != nil {
		return err
	}
	m.resultBytes += size
	change.SchemaVersion = 1
	m.report.Changes = append(m.report.Changes, change)
	return nil
}
func objectSize(o Object) int {
	return 512 + len(o.Identity) + len(o.Scope) + len(o.Value) + len(o.Location.SourceKey) + len(o.Location.Source.Name) + len(o.Location.ASTIDs)*8
}
func copyObject(object Object) *Object { return &object }
func confidence(objects ...Object) string {
	for _, object := range objects {
		if object.Uninterpreted {
			return "uninterpreted-source"
		}
	}
	if len(objects) > 0 && objects[0].Kind == SourceFile {
		return "source-bytes"
	}
	return "projected-content"
}
func changeObject(change Change) *Object {
	if change.Before != nil {
		return change.Before
	}
	return change.After
}
func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func snapshotIdentity(e *entity) string {
	location := e.object.Location
	return tuple(location.SourceKey, location.Source.Digest, strconv.FormatUint(uint64(location.SemanticID), 10), strconv.Itoa(location.Span.Start.Offset), strconv.Itoa(location.Span.End.Offset))
}
func unpaired(group []*entity) []*entity {
	var out []*entity
	for _, e := range group {
		if !e.paired {
			out = append(out, e)
		}
	}
	return out
}
func (m *matcher) alignUnique(a, b []*entity, keyFor func(*entity) string) error {
	old, new := map[string][]*entity{}, map[string][]*entity{}
	for _, entry := range []struct {
		items  []*entity
		groups map[string][]*entity
	}{{a, old}, {b, new}} {
		for _, e := range entry.items {
			if err := m.meter.spend(len(e.object.Value) + 1); err != nil {
				return err
			}
			key := keyFor(e)
			entry.groups[key] = append(entry.groups[key], e)
		}
	}
	for _, e := range a {
		key := keyFor(e)
		if len(old[key]) == 1 && len(new[key]) == 1 {
			other := new[key][0]
			e.paired = true
			other.paired = true
			m.pairs = append(m.pairs, &pair{before: e, after: other})
		}
	}
	return nil
}
