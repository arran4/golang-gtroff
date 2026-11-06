package extract

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/query"
	"github.com/arran4/golang-gtroff/source"
)

type builder struct {
	edgeByID     map[source.EdgeID]source.Edge
	inherited    map[source.ID][]Heading
	set          *source.Set
	models       map[source.ID]*man.Document
	infos        map[source.ID]query.SourceInfo
	meter        *meter
	options      Options
	nodes, bytes int
	report       *Report
	edges        map[source.ID][]source.Edge
	diagnostics  map[string]bool
}

func fromQuery(ctx context.Context, set *source.Set, selected *query.Result, o Options, spent int) (report *Report, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	report = newReport(selected, o)
	var accounting *meter
	defer func() {
		if accounting != nil {
			report.WorkUnits = o.MaxWork - accounting.remaining
		}
		if err == nil {
			err = ctx.Err()
		}
		if err != nil {
			report.Error = err.Error()
		}
		sortDiagnostics(report.Diagnostics)
	}()
	o, err = defaults(o)
	if err != nil {
		return report, err
	}
	if set == nil || selected == nil || selected.SchemaVersion != 1 || selected.View != query.Syntax && selected.View != query.Semantic {
		return report, ErrSnapshot
	}
	if o.HTML && selected.View != query.Semantic {
		return report, fmt.Errorf("%w: HTML requires semantic selection", ErrOptions)
	}
	b := &builder{set: set, models: map[source.ID]*man.Document{}, infos: map[source.ID]query.SourceInfo{}, meter: &meter{ctx, o.MaxWork}, options: o, report: report, edges: map[source.ID][]source.Edge{}, edgeByID: map[source.EdgeID]source.Edge{}, inherited: map[source.ID][]Heading{}, diagnostics: map[string]bool{}}
	accounting = b.meter
	if err = b.meter.spend(spent + 1); err != nil {
		return report, err
	}
	initialDiagnostics := report.Diagnostics
	report.Diagnostics = nil
	for _, d := range initialDiagnostics {
		b.addDiagnostic(d)
	}
	for _, d := range set.Diagnostics() {
		b.addDiagnostic(d)
	}
	for _, e := range set.Edges() {
		b.edges[e.From] = append(b.edges[e.From], e)
		b.edgeByID[e.ID] = e
	}
	previousIndex := 0
	for _, match := range selected.Matches {
		if len(report.Records) >= o.MaxRecords {
			return report, ErrLimit
		}
		if err = b.meter.spend(1); err != nil {
			return report, err
		}
		if match.SchemaVersion != 1 || match.View != selected.View || match.Index <= previousIndex || match.ID == "" {
			return report, ErrSnapshot
		}
		previousIndex = match.Index
		var record Record
		record, err = b.record(match)
		if err != nil {
			return report, err
		}
		record.Index = len(report.Records) + 1
		size := recordSize(record)
		if size > o.MaxBytes-b.bytes {
			return report, ErrLimit
		}
		b.bytes += size
		report.Records = append(report.Records, record)
	}
	report.Complete = true
	if !selected.Complete || selected.Error != "" {
		return report, ErrIncomplete
	}
	if o.Strict {
		if report.InputError != "" || len(report.Diagnostics) > 0 {
			return report, ErrStrict
		}
		for _, r := range report.Records {
			if !r.Source.Complete {
				return report, ErrStrict
			}
		}
	}
	return report, nil
}
func (b *builder) info(id source.ID) (query.SourceInfo, error) {
	if info, ok := b.infos[id]; ok {
		return info, nil
	}
	file := b.set.File(id)
	if file == nil {
		return query.SourceInfo{}, ErrSnapshot
	}
	data := file.AST().Source()
	if err := b.meter.spend(len(data) + len(file.Name()) + 1); err != nil {
		return query.SourceInfo{}, err
	}
	var walkErr error
	file.AST().Root().Walk(func(*groff.ASTNode) groff.WalkAction {
		if b.nodes >= b.options.MaxNodes {
			walkErr = ErrLimit
			return groff.WalkStop
		}
		b.nodes++
		if err := b.meter.spend(1); err != nil {
			walkErr = err
			return groff.WalkStop
		}
		return groff.WalkContinue
	})
	if walkErr != nil {
		return query.SourceInfo{}, walkErr
	}
	digest := sha256.Sum256([]byte(data))
	info := query.SourceInfo{ID: id, Name: file.Name(), ParentEdge: file.ParentEdge(), Depth: file.Depth(), Digest: fmt.Sprintf("%x", digest), Complete: file.AST().SourceComplete()}
	b.infos[id] = info
	return info, nil
}
func (b *builder) model(id source.ID) (*man.Document, error) {
	if model := b.models[id]; model != nil {
		return model, nil
	}
	file := b.set.File(id)
	if err := b.meter.spend(len(file.AST().Source()) + 1); err != nil {
		return nil, err
	}
	model, err := man.Build(file.AST(), man.Options{MaxNodes: b.options.MaxNodes, MaxBytes: len(file.AST().Source()) + 1})
	if err != nil {
		if errors.Is(err, man.ErrLimit) {
			err = errors.Join(ErrLimit, err)
		}
		return nil, err
	}
	b.models[id] = model
	for _, d := range model.Diagnostics {
		b.addDiagnostic(source.Diagnostic{Source: id, Code: "semantic", Diagnostic: d})
	}
	return model, nil
}
func (b *builder) record(match query.Match) (Record, error) {
	info, err := b.info(match.Source.ID)
	if err != nil {
		return Record{}, err
	}
	if match.Source != info {
		return Record{}, ErrSnapshot
	}
	file := b.set.File(info.ID)
	if match.Span.Start.Offset < 0 || match.Span.End.Offset < match.Span.Start.Offset || match.Span.End.Offset > len(file.AST().Source()) {
		return Record{}, ErrSnapshot
	}
	if match.Span.End.Offset-match.Span.Start.Offset > b.options.MaxBytes-b.bytes-512 {
		return Record{}, ErrLimit
	}
	r := Record{SchemaVersion: 1, QueryIndex: match.Index, ID: match.ID, View: match.View, Kind: match.Kind, Source: info, Span: match.Span, SemanticID: match.SemanticID}
	var model *man.Document
	var node *man.Node
	switch {
	case match.Kind == query.Source:
		if match.SemanticID != 0 || len(match.ASTIDs) != 0 || match.Span != file.AST().Root().Span() {
			return r, ErrSnapshot
		}
		r.Name = file.Name()
		r.Text = file.AST().Root().Text()
	case match.View == query.Syntax:
		if match.SemanticID != 0 || len(match.ASTIDs) != 1 {
			return r, ErrSnapshot
		}
		syntax := file.AST().Node(match.ASTIDs[0])
		if syntax == nil || syntax.Span() != match.Span || query.Kind(syntax.Kind()) != match.Kind {
			return r, ErrSnapshot
		}
		r.ASTIDs = append([]groff.NodeID(nil), match.ASTIDs...)
		r.Name = syntax.Attributes().Name
		if syntax.Kind() == groff.ASTSection || syntax.Kind() == groff.ASTSubsection {
			r.Name = syntax.Attributes().Heading
		}
		r.Text = syntax.Text()
	default:
		model, err = b.model(info.ID)
		if err != nil {
			return r, err
		}
		node = model.Node(match.SemanticID)
		if node == nil {
			return r, ErrSnapshot
		}
		kind := query.Kind(node.Kind)
		if node.Kind == man.Item {
			kind = query.Definition
		}
		span := node.Origin.Span
		ids := node.Origin.ASTIDs
		if match.Kind == query.Term {
			if node.Kind != man.Item || len(node.Tag) == 0 {
				return r, ErrSnapshot
			}
			kind = query.Term
			span, ids = tagOrigin(node.Tag)
		}
		if kind != match.Kind || span != match.Span || !slices.Equal(ids, match.ASTIDs) {
			return r, ErrSnapshot
		}
		r.ASTIDs = append([]groff.NodeID(nil), ids...)
		r.Semantic = true
		metadata := model.Metadata
		r.Metadata = &metadata
		if model.MetadataOrigin != nil {
			origin := *model.MetadataOrigin
			origin.ASTIDs = append([]groff.NodeID(nil), origin.ASTIDs...)
			r.MetadataOrigin = &origin
		}
		state := node.State
		state.Indents = append([]string(nil), state.Indents...)
		r.Context = &state
		parents, headingErr := b.inheritedHeadings(info.ID)
		if headingErr != nil {
			return r, headingErr
		}
		r.Headings = append(append([]Heading(nil), parents...), headingPath(model, node, info.ID)...)
		if err = b.meter.spend(len(r.Headings)); err != nil {
			return r, err
		}
		r.Name = ""
		if len(node.Origin.ASTIDs) > 0 {
			if syntax := file.AST().Node(node.Origin.ASTIDs[0]); syntax != nil {
				r.Name = syntax.Attributes().Name
			}
		}
		if node.Kind == man.Item {
			r.TermRuns = cloneRuns(node.Tag)
			r.Term = runsText(node.Tag)
			r.Name = r.Term
		}
		if node.Kind == man.Section || node.Kind == man.Subsection {
			r.Name = node.Heading
		}
		if match.Kind == query.Term {
			r.Text = r.Term
			r.Unsupported = unsupportedRuns(node.Tag)
		} else {
			r.Blocks, err = b.blocks(model, node.ID)
			if err != nil {
				return r, err
			}
			selection := man.Selection{Nodes: r.Blocks}
			r.Text = selection.Text(man.TextOptions{})
			for _, n := range r.Blocks {
				r.Unsupported = r.Unsupported || n.Kind == man.Unsupported || unsupportedRuns(n.Tag) || unsupportedRuns(n.Runs)
			}
			if node.Kind == man.Item {
				body := append([]man.Node(nil), r.Blocks...)
				body[0].Tag = nil
				r.Description = (&man.Selection{Nodes: body}).Text(man.TextOptions{})
			}
		}
	}
	if r.Span.Start.Offset < 0 || r.Span.End.Offset < r.Span.Start.Offset || r.Span.End.Offset > len(file.AST().Source()) {
		return r, ErrSnapshot
	}
	r.Raw = file.AST().Source()[r.Span.Start.Offset:r.Span.End.Offset]
	if err = b.meter.spend(len(r.Raw) + len(r.Text) + len(r.Description) + len(r.Term)); err != nil {
		return r, err
	}
	r.NormalizedText = strings.Join(strings.Fields(r.Text), " ")
	for _, edge := range b.edges[info.ID] {
		if err = b.meter.spend(1); err != nil {
			return r, err
		}
		if edge.Span.Start.Offset >= r.Span.Start.Offset && edge.Span.Start.Offset < r.Span.End.Offset {
			r.Includes = append(r.Includes, edge)
		}
	}
	if r.Unsupported {
		b.addDiagnostic(source.Diagnostic{Source: info.ID, Code: "unsupported-content", Diagnostic: groff.Diagnostic{Severity: "warning", Position: r.Span.Start, Message: "selected content includes unsupported optional-man semantics; literal source and blocks are retained"}})
	}
	if b.options.HTML {
		if model == nil {
			return r, fmt.Errorf("%w: HTML requires a semantic content node, not a source wrapper", ErrOptions)
		}
		r.HTML, err = b.render(model, node, &r)
		if err != nil {
			return r, err
		}
	}
	for _, d := range b.report.Diagnostics {
		if err = b.meter.spend(1); err != nil {
			return r, err
		}
		if d.Source == info.ID && d.Position.Offset >= r.Span.Start.Offset && d.Position.Offset < r.Span.End.Offset {
			r.Diagnostics = append(r.Diagnostics, d)
		}
	}
	sortDiagnostics(r.Diagnostics)
	return r, nil
}
func (b *builder) blocks(model *man.Document, id man.ID) ([]man.Node, error) {
	var out []man.Node
	materialized := 0
	stack := []man.ID{id}
	for len(stack) > 0 {
		next := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		node := model.Node(next)
		if err := b.meter.spend(1 + len(node.Runs) + len(node.Tag) + len(node.Children)); err != nil {
			return nil, err
		}
		materialized += nodeSize(*node)
		if materialized > b.options.MaxBytes-b.bytes {
			return nil, ErrLimit
		}
		out = append(out, cloneNode(*node))
		for i := len(node.Children) - 1; i >= 0; i-- {
			stack = append(stack, node.Children[i])
		}
	}
	return out, nil
}
func cloneRuns(runs []man.Run) []man.Run {
	out := append([]man.Run(nil), runs...)
	for i := range out {
		out[i].Origin.ASTIDs = append([]groff.NodeID(nil), out[i].Origin.ASTIDs...)
	}
	return out
}
func cloneNode(node man.Node) man.Node {
	node.Children = append([]man.ID(nil), node.Children...)
	node.State.Indents = append([]string(nil), node.State.Indents...)
	node.Origin.ASTIDs = append([]groff.NodeID(nil), node.Origin.ASTIDs...)
	node.Runs = cloneRuns(node.Runs)
	node.Tag = cloneRuns(node.Tag)
	return node
}
func runsText(runs []man.Run) string {
	var text strings.Builder
	for _, run := range runs {
		text.WriteString(run.Text)
	}
	return text.String()
}
func unsupportedRuns(runs []man.Run) bool {
	for _, run := range runs {
		if run.Unsupported {
			return true
		}
	}
	return false
}
func tagOrigin(runs []man.Run) (groff.Span, []groff.NodeID) {
	span := runs[0].Origin.Span
	var ids []groff.NodeID
	seen := map[groff.NodeID]bool{}
	for _, run := range runs {
		if run.Origin.Span.Start.Offset < span.Start.Offset {
			span.Start = run.Origin.Span.Start
		}
		if run.Origin.Span.End.Offset > span.End.Offset {
			span.End = run.Origin.Span.End
		}
		for _, id := range run.Origin.ASTIDs {
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	return span, ids
}
func headingPath(model *man.Document, node *man.Node, id source.ID) []Heading {
	var headings []Heading
	for current := node; current != nil; current = model.Node(current.Parent) {
		if current.Kind == man.Section || current.Kind == man.Subsection {
			headings = append(headings, Heading{id, current.ID, current.Heading, current.Origin.Span})
		}
	}
	slices.Reverse(headings)
	return headings
}
func diagKey(d source.Diagnostic) string {
	return fmt.Sprintf("%d:%d:%s:%s", d.Source, d.Position.Offset, d.Severity, d.Message)
}
func (b *builder) addDiagnostic(d source.Diagnostic) {
	key := diagKey(d)
	if !b.diagnostics[key] {
		b.diagnostics[key] = true
		b.report.Diagnostics = append(b.report.Diagnostics, d)
	}
}
func sortDiagnostics(ds []source.Diagnostic) {
	sort.SliceStable(ds, func(i, j int) bool {
		if ds[i].Source != ds[j].Source {
			return ds[i].Source < ds[j].Source
		}
		if ds[i].Position.Offset != ds[j].Position.Offset {
			return ds[i].Position.Offset < ds[j].Position.Offset
		}
		return ds[i].Message < ds[j].Message
	})
}
func recordSize(r Record) int {
	size := 512 + len(r.ID) + len(r.Name) + len(r.Source.Name) + len(r.Text) + len(r.NormalizedText) + len(r.Raw) + len(r.Term) + len(r.Description) + len(r.HTML) + len(r.ASTIDs)*8
	for _, node := range r.Blocks {
		size += nodeSize(node)
	}
	if r.Metadata != nil {
		size += len(r.Metadata.Title) + len(r.Metadata.Section) + len(r.Metadata.Date) + len(r.Metadata.Source) + len(r.Metadata.Manual)
	}
	if r.MetadataOrigin != nil {
		size += 128 + len(r.MetadataOrigin.ASTIDs)*8
	}
	if r.Context != nil {
		size += stateSize(*r.Context)
	}
	for _, run := range r.TermRuns {
		size += runSize(run)
	}
	for _, h := range r.Headings {
		size += 128 + len(h.Name)
	}
	for _, d := range r.Diagnostics {
		size += 128 + len(d.Message)
	}
	for _, e := range r.Includes {
		size += 256 + len(e.Target)
	}
	return size
}

func (b *builder) inheritedHeadings(id source.ID) ([]Heading, error) {
	if headings, ok := b.inherited[id]; ok {
		return headings, nil
	}
	file := b.set.File(id)
	if file.ParentEdge() == 0 {
		b.inherited[id] = nil
		return nil, nil
	}
	edge, ok := b.edgeByID[file.ParentEdge()]
	if !ok || edge.From >= id {
		return nil, ErrSnapshot
	}
	parents, err := b.inheritedHeadings(edge.From)
	if err != nil {
		return nil, err
	}
	if _, err = b.info(edge.From); err != nil {
		return nil, err
	}
	model, err := b.model(edge.From)
	if err != nil {
		return nil, err
	}
	if err = b.meter.spend(len(model.Nodes)*4 + 1); err != nil {
		return nil, err
	}
	owner, err := model.SelectAST(edge.Node)
	if err != nil {
		return nil, err
	}
	headings := append(append([]Heading(nil), parents...), headingPath(model, model.Node(owner.Roots[0]), edge.From)...)
	b.inherited[id] = headings
	return headings, nil
}

func nodeSize(node man.Node) int {
	size := 512 + len(node.Heading) + len(node.Value) + len(node.Children)*8 + len(node.Origin.ASTIDs)*8 + stateSize(node.State)
	for _, runs := range [][]man.Run{node.Tag, node.Runs} {
		for _, run := range runs {
			size += runSize(run)
		}
	}
	return size
}
func runSize(run man.Run) int {
	return 128 + len(run.Text) + len(run.Link) + len(run.Reference) + len(run.Origin.ASTIDs)*8
}
func stateSize(state man.State) int {
	size := len(state.TagIndent) + len(state.Link) + len(state.ParagraphDistance)
	for _, indent := range state.Indents {
		size += len(indent)
	}
	return size
}
