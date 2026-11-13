package docdiff

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/extract"
	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/query"
	"github.com/arran4/golang-gtroff/source"
)

type segment struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}
type entity struct {
	object   Object
	segments []segment
	parent   string
	position int
	paired   bool
}
type index struct {
	sourceKeys  map[source.ID]string
	entities    []*entity
	bySource    map[source.ID]map[groff.NodeID]*entity
	meter       *meter
	options     Options
	data        *int
	diagnostics []Diagnostic
	side        string
}

func (i *index) add(kind Kind, identity, scope string, location Location, value string, uninterpreted bool) (*entity, error) {
	if len(i.entities) >= i.options.MaxObjects {
		return nil, ErrLimit
	}
	if err := i.meter.spend(len(identity) + len(scope) + len(value) + 1); err != nil {
		return nil, err
	}
	e := &entity{object: Object{Index: len(i.entities) + 1, Kind: kind, Identity: identity, Scope: scope, Location: location, Value: value, Uninterpreted: uninterpreted}, position: len(i.entities)}
	i.entities = append(i.entities, e)
	return e, nil
}
func buildIndex(s *inspected, side string, o Options, m *meter, data *int) (*index, error) {
	i := &index{bySource: map[source.ID]map[groff.NodeID]*entity{}, sourceKeys: s.keys, meter: m, options: o, data: data, side: side}
	if o.View == Source {
		for _, info := range s.info.Files {
			file := s.input.Sources.File(info.ID)
			location := Location{Source: info, SourceKey: s.keys[info.ID], Span: file.AST().Root().Span(), ASTIDs: []groff.NodeID{file.AST().Root().ID()}}
			if _, err := i.add(SourceFile, s.keys[info.ID], "source-set", location, file.AST().Source(), false); err != nil {
				return i, err
			}
		}
		return i, i.finish()
	}
	if m.remaining < 1 {
		return i, ErrLimit
	}
	// One existing query/extraction projection per revision supplies complete
	// file-local man models, source contributors, include ancestry and diagnostics.
	projected, err := extract.Select(m.ctx, s.input.Sources, query.Select(query.Document), extract.Options{MaxNodes: o.MaxNodes, MaxRecords: 10000, MaxBytes: o.MaxDataBytes, MaxWork: m.remaining})
	if chargeErr := m.spend(projected.WorkUnits); chargeErr != nil {
		return i, chargeErr
	}
	for _, d := range projected.Diagnostics {
		i.diagnostics = append(i.diagnostics, Diagnostic{side, d.Code, d.Message, d.Source, d.Position})
	}
	if err != nil {
		if errors.Is(err, extract.ErrLimit) || errors.Is(err, query.ErrLimit) {
			err = errors.Join(ErrLimit, err)
		}
		return i, err
	}
	sort.Slice(projected.Records, func(a, b int) bool { return projected.Records[a].Source.ID < projected.Records[b].Source.ID })
	for _, record := range projected.Records {
		if err := i.file(s, record); err != nil {
			return i, err
		}
	}
	return i, i.finish()
}
func (i *index) file(s *inspected, r extract.Record) error {
	sourceKey := s.keys[r.Source.ID]
	base := sourceKey
	file := s.input.Sources.File(r.Source.ID)
	if file.ParentEdge() != 0 {
		for _, edge := range s.info.Includes {
			if err := i.meter.spend(1); err != nil {
				return err
			}
			if edge.ID == file.ParentEdge() {
				if owner := i.bySource[edge.From][edge.Node]; owner != nil {
					base = tuple(sourceKey, owner.parent)
				}
				break
			}
		}
	}
	nodes := map[man.ID]man.Node{}
	for _, node := range r.Blocks {
		nodes[node.ID] = node
	}
	locations := func(node man.Node) Location {
		return Location{Source: r.Source, SourceKey: sourceKey, Span: node.Origin.Span, ASTIDs: append([]groff.NodeID(nil), node.Origin.ASTIDs...), SemanticID: node.ID}
	}
	root := nodes[1]
	rootLocation := locations(root)
	if r.Metadata != nil && *r.Metadata != (man.Metadata{}) {
		if r.MetadataOrigin == nil {
			return fmt.Errorf("%w: metadata origin missing from source-backed projection", ErrInput)
		}
		location := Location{Source: r.Source, SourceKey: sourceKey, Span: r.MetadataOrigin.Span, ASTIDs: append([]groff.NodeID(nil), r.MetadataOrigin.ASTIDs...)}
		value, _ := json.Marshal(r.Metadata)
		uncertain := false
		for _, diagnostic := range i.diagnostics {
			if err := i.meter.spend(1); err != nil {
				return err
			}
			if diagnostic.Source == r.Source.ID && diagnostic.Code == "semantic" && diagnostic.Position.Offset >= location.Span.Start.Offset && diagnostic.Position.Offset < location.Span.End.Offset {
				uncertain = true
			}
		}
		if _, err := i.add(Metadata, "$metadata", sourceKey, location, string(value), uncertain); err != nil {
			return err
		}
	}
	owners := map[man.ID]*entity{}
	var body *entity
	sourceOwners := map[groff.NodeID]*entity{}
	i.bySource[r.Source.ID] = sourceOwners
	for _, node := range r.Blocks {
		if err := i.meter.spend(1); err != nil {
			return err
		}
		parent := owners[node.Parent]
		scope := base
		if parent != nil {
			scope = parent.parent
		}
		kind := Kind("")
		identity := ""
		switch node.Kind {
		case man.Section:
			kind = Section
			identity = flow(node.Heading)
			scope = base
		case man.Subsection:
			kind = Subsection
			identity = flow(node.Heading)
		case man.Item:
			kind = Definition
			identity = flow(runText(node.Tag))
		}
		owner := parent
		if kind != "" {
			var err error
			owner, err = i.add(kind, identity, scope, locations(node), "", false)
			if err != nil {
				return err
			}
			owner.parent = tuple(scope, string(kind), identity)
		}
		if kind == Definition && owner != nil {
			tagNode := node
			tagNode.Runs = node.Tag
			if _, err := i.contentText(s, r, tagNode, owner.parent, owner); err != nil {
				return err
			}
		}
		owners[node.ID] = owner
		for _, id := range node.Origin.ASTIDs {
			sourceOwners[id] = owner
		}
		if node.Kind == man.Paragraph || node.Kind == man.Preformatted || node.Kind == man.Unsupported {
			if node.Kind == man.Unsupported {
				isInclude := false
				name := "uninterpreted"
				for _, id := range node.Origin.ASTIDs {
					if syntax := file.AST().Node(id); syntax != nil {
						if syntax.Attributes().Name == ".so" || syntax.Attributes().Name == ".mso" {
							isInclude = true
						}
						if syntax.Attributes().Name != "" {
							name = syntax.Attributes().Name
							break
						}
					}
				}
				if !isInclude {
					raw := ""
					a, b := node.Origin.Span.Start.Offset, node.Origin.Span.End.Offset
					if a >= 0 && b >= a && b <= len(file.AST().Source()) {
						raw = file.AST().Source()[a:b]
					}
					if _, err := i.add(Opaque, name, scope, locations(node), raw, true); err != nil {
						return err
					}
					i.diagnostics = append(i.diagnostics, Diagnostic{i.side, "uninterpreted-region", "unsupported region is compared as source bytes, not interpreted meaning", r.Source.ID, node.Origin.Span.Start})
				}
				continue
			}
			if owner == nil && node.Kind == man.Paragraph && flow(runText(node.Runs)) == "" {
				meaningful := false
				for _, run := range node.Runs {
					meaningful = meaningful || run.Unsupported || run.Link != "" || run.Reference != ""
				}
				if !meaningful {
					continue
				}
			}
			if owner == nil {
				if body == nil {
					var err error
					body, err = i.add(Preamble, "$preamble", base, rootLocation, "", false)
					if err != nil {
						return err
					}
					body.parent = tuple(base, "preamble")
				}
				owner = body
			}
			text, opaqueErr := i.contentText(s, r, node, scope, owner)
			if opaqueErr != nil {
				return opaqueErr
			}
			segmentKind := "flow"
			if node.Kind == man.Preformatted {
				segmentKind = "preformatted"
			} else {
				text = flow(text)
			}
			if err := i.meter.spend(len(text) + 1); err != nil {
				return err
			}
			owner.segments = append(owner.segments, segment{segmentKind, text})
		}
		if err := i.references(node, r, scope, owner); err != nil {
			return err
		}

	}
	return nil
}
func (i *index) finish() error {
	for _, entity := range i.entities {
		if len(entity.segments) > 0 {
			value, _ := json.Marshal(collapseSegments(entity.segments))
			entity.object.Value = string(value)
			entity.segments = nil
		}
		size := len(entity.object.Value) + len(entity.object.Identity) + len(entity.object.Scope) + 512
		if size > i.options.MaxDataBytes-*i.data {
			return ErrLimit
		}
		*i.data += size
		if err := i.meter.spend(size); err != nil {
			return err
		}
		digest := sha256.Sum256([]byte(entity.object.Value))
		entity.object.Digest = fmt.Sprintf("%x", digest)
	}
	return nil
}
func flow(text string) string { return strings.Join(strings.Fields(text), " ") }
func runText(runs []man.Run) string {
	var text strings.Builder
	for _, run := range runs {
		text.WriteString(run.Text)
	}
	return text.String()
}

func collapseSegments(segments []segment) []segment {
	var out []segment
	var pending strings.Builder
	flush := func() {
		if text := flow(pending.String()); text != "" {
			out = append(out, segment{"flow", text})
		}
		pending.Reset()
	}
	for _, seg := range segments {
		if seg.Kind == "flow" {
			pending.WriteString(seg.Text)
			pending.WriteByte(' ')
		} else {
			flush()
			out = append(out, seg)
		}
	}
	flush()
	return out
}
func (i *index) references(node man.Node, r extract.Record, scope string, owner *entity) error {
	if owner != nil {
		scope = owner.parent
	}
	for _, runs := range [][]man.Run{node.Runs, node.Tag} {
		var target, kind string
		var text strings.Builder
		var origin man.Origin
		seenOrigins := map[groff.NodeID]bool{}
		flush := func() error {
			if target == "" {
				return nil
			}
			value, _ := json.Marshal(struct{ Kind, Target string }{kind, target})
			location := Location{Source: r.Source, SourceKey: i.sourceKeys[r.Source.ID], Span: origin.Span, ASTIDs: append([]groff.NodeID(nil), origin.ASTIDs...), SemanticID: node.ID}
			_, err := i.add(Reference, kind+" "+flow(text.String()), scope, location, string(value), false)
			return err
		}
		for _, run := range runs {
			if err := i.meter.spend(len(run.Text) + 1); err != nil {
				return err
			}
			next, nextKind := run.Link, "url"
			if run.Reference != "" {
				next = run.Reference
				nextKind = "manual"
			}
			if target != "" && next == "" && strings.TrimSpace(run.Text) == "" {
				text.WriteString(run.Text)
				continue
			}
			if next != target || nextKind != kind {
				if err := flush(); err != nil {
					return err
				}
				target, kind = next, nextKind
				text.Reset()
				origin = run.Origin
				origin.ASTIDs = append([]groff.NodeID(nil), origin.ASTIDs...)
				seenOrigins = map[groff.NodeID]bool{}
				for _, id := range origin.ASTIDs {
					seenOrigins[id] = true
				}
			}
			if target != "" {
				text.WriteString(run.Text)
				if run.Origin.Span.End.Offset > origin.Span.End.Offset {
					origin.Span.End = run.Origin.Span.End
				}
				for _, id := range run.Origin.ASTIDs {
					if !seenOrigins[id] {
						seenOrigins[id] = true
						origin.ASTIDs = append(origin.ASTIDs, id)
					}
				}
			}
		}
		if err := flush(); err != nil {
			return err
		}
	}
	return nil
}
func (i *index) contentText(s *inspected, r extract.Record, node man.Node, scope string, owner *entity) (string, error) {
	var text strings.Builder
	seen := map[int]bool{}
	file := s.input.Sources.File(r.Source.ID)
	for _, run := range node.Runs {
		if !run.Unsupported {
			text.WriteString(run.Text)
			continue
		}
		owner.object.Uninterpreted = true
		text.WriteString("[uninterpreted]")
		offset := run.Origin.Span.Start.Offset
		if seen[offset] {
			continue
		}
		seen[offset] = true
		a, b := offset, run.Origin.Span.End.Offset
		if a < 0 || b < a || b > len(file.AST().Source()) {
			return "", ErrInput
		}
		location := Location{Source: r.Source, SourceKey: s.keys[r.Source.ID], Span: run.Origin.Span, ASTIDs: append([]groff.NodeID(nil), run.Origin.ASTIDs...), SemanticID: node.ID}
		if _, err := i.add(Opaque, "inline-unsupported", scope, location, file.AST().Source()[a:b], true); err != nil {
			return "", err
		}
		i.diagnostics = append(i.diagnostics, Diagnostic{i.side, "uninterpreted-inline", "unsupported inline source is preserved; its runtime meaning is unassessed", r.Source.ID, run.Origin.Span.Start})
	}
	return text.String(), nil
}
