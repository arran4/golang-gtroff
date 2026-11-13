package analyze

import (
	"fmt"
	"strings"

	groff "github.com/arran4/golang-gtroff"
)

// RuleInfo describes an always-enabled built-in detector. Custom declarative
// rules supplement these IDs and cannot replace their capability classifications.
type RuleInfo struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

func BuiltinRules() []RuleInfo {
	return []RuleInfo{
		{"builtin.control", "Physical control spellings, parser route and capability evidence"},
		{"builtin.text", "Physical text, blank lines and comments"},
		{"builtin.escape", "Parsed inline escape families and references"},
		{"builtin.include", "Source references and actual include-resolution state"},
		{"builtin.definition", "Register/string/macro definition syntax without evaluation"},
		{"builtin.conditional", "Conditional/loop syntax requiring interpretation"},
		{"builtin.programmatic", "Other stateful flow/trap/diversion/environment syntax"},
		{"builtin.side-effect", "Possible execution/filesystem/device requests, never executed"},
		{"builtin.preprocessor", "Captured tbl/eqn/pic/refer source requiring interpretation"},
		{"builtin.reference", "Explicit manual/URL/mail references as data"},
		{"builtin.structure", "Source groups, nesting and opaque capture boundaries"},
	}
}
func (r *runner) builtins(c nodeContext) error {
	n, a := c.node, c.attrs
	emit := func(rule, feature, name, reason string, requires []string, target string) error {
		return r.emit(c, "builtin."+rule, feature, name, reason, requires, target)
	}
	if n.Kind() == groff.ASTBlock {
		if a.Opaque {
			if err := emit("structure", "opaque", a.Name, "Original capture retained; stored controls are not executed or modeled", nil, ""); err != nil {
				return err
			}
		}
		if n.Parent() != nil {
			for p := n.Parent(); p != nil; p = p.Parent() {
				if p.Kind() == groff.ASTBlock {
					return emit("structure", "nesting", a.Name, "Nested syntax boundary; this is source structure, not runtime expansion", nil, "")
				}
			}
		}
	}
	if n.Kind() == groff.ASTSection || n.Kind() == groff.ASTSubsection {
		if err := emit("structure", string(n.Kind()), a.Heading, "Source heading group preserves original order and positions", nil, ""); err != nil {
			return err
		}
	}
	if n.Kind() == groff.ASTInline && c.inline.Kind != groff.InlineText {
		var requires []string
		if c.capability.Semantics == Unsupported || c.capability.Semantics == NotAssessed || c.capability.Semantics == PartiallyModeled {
			requires = append(requires, "advanced-interpretation")
		}
		if c.inline.Kind == groff.InlineStringRef || c.inline.Kind == groff.InlineRegisterRef {
			requires = append(requires, "evaluation")
		}
		reason := "Inline grammar parsed separately from expansion and rendering"
		if c.capability.Semantics == Unsupported {
			reason += "; bounded man interpretation leaves this escape literal"
		}
		return emit("escape", "escape", string(c.inline.Kind), reason, requires, c.inline.Name)
	}
	if n.Kind() != groff.ASTLine {
		return nil
	}
	if a.TokenKind == groff.KindComment {
		return emit("text", "comment", "", "Parsed comment retained as source; it is not executed", nil, "")
	}
	if a.Name == "" {
		feature := "text"
		if strings.TrimSpace(n.Raw()) == "" {
			feature = "blank"
		}
		return emit("text", feature, "", "Physical source line retained independently of display interpretation", nil, "")
	}
	reason := "Parsed syntax is independent of semantic support and successful rendering"
	switch c.capability.Syntax {
	case Opaque:
		reason = "Control-looking source is stored in an opaque region; no invocation is interpreted"
	case Unknown:
		reason = "Control spelling is absent from the parser catalogue; original syntax remains visible"
	case Recognized:
		reason = "Catalogue spelling recognized, but this snapshot has malformed or package-incompatible syntax"
	}
	switch c.capability.Semantics {
	case Modeled:
		reason += "; bounded man projection accepted this construct without an unresolved diagnostic"
	case PartiallyModeled:
		reason += "; man interpretation has unresolved details at this source position"
	case Unsupported:
		reason += "; man projection retains unsupported literal source"
	case NotAssessed:
		reason += "; no active semantic contributor was assessed for this context"
	}
	var controlRequires []string
	if c.capability.Semantics != Modeled {
		controlRequires = []string{"advanced-interpretation"}
	}
	if err := emit("control", "control", a.Name, reason, controlRequires, ""); err != nil {
		return err
	}
	first := ""
	if len(a.Arguments) > 0 {
		first = a.Arguments[0].Value
	}
	switch a.Name {
	case ".so", ".mso":
		include := c
		state := "opaque"
		reason := "Stored include-looking source is not a load instruction"
		if edge, ok := r.edges[c.file.ID()][n.ID()]; ok {
			state = edge.State
			first = edge.Target
			reason = "Include directive retained separately from source loading; no macro execution implied"
			if edge.State == "incomplete" && edge.To != 0 {
				state = string(PartiallyResolved)
			}
		} else if c.active {
			state = "unresolved"
			reason = "Recognized include has no source-resolution edge in this snapshot"
		}
		include.capability.Resolution = state
		return r.emit(include, "builtin.include", "include", a.Name, reason, []string{"source-resolution"}, first)
	case ".de", ".am", ".de1", ".dei", ".dei1", ".am1", ".ami", ".ami1":
		definition := c
		reason = "Macro definition header/body preserved; invocation and expansion are not executed"
		if p := n.Parent(); c.active && p != nil && p.Kind() == groff.ASTBlock && p.Attributes().Name == a.Name && !p.Attributes().Closed {
			definition.capability.Syntax = Recognized
			reason += "; capture boundary is incomplete or unresolved"
			definition.capability.Resolution = "unresolved-delimiter"
		}
		return r.emit(definition, "builtin.definition", "macro-definition", a.Name, reason, []string{"evaluation"}, first)
	case ".ds", ".ds1", ".as", ".as1":
		return emit("definition", "string-definition", a.Name, "String storage/update syntax parsed without expanding references", []string{"evaluation"}, first)
	case ".nr", ".nrf", ".rr", ".rnn", ".aln", ".af":
		return emit("definition", "register-definition", a.Name, "Register update/removal syntax retained without arithmetic evaluation", []string{"evaluation"}, first)
	case ".als", ".rm", ".rn":
		return emit("definition", "definition-update", a.Name, "Definition alias/removal/rename syntax requires stateful interpretation", []string{"evaluation"}, first)
	case ".if", ".ie", ".el", ".while":
		return emit("conditional", "conditional", a.Name, "Conditional/loop syntax is present; branches are not executed or assumed true", []string{"evaluation"}, first)
	case ".return", ".shift", ".break", ".continue", ".nop", ".do", ".it", ".itc", ".wh", ".ch", ".di", ".da", ".ev":
		return emit("programmatic", "programmatic", a.Name, "Stateful flow, trap, diversion or environment syntax requires interpretation", []string{"evaluation"}, first)
	case ".sy", ".pso", ".open", ".opena", ".write", ".writem", ".close", ".pi", ".cf":
		return emit("side-effect", "side-effect-syntax", a.Name, "Would request a command, file or device effect in an interpreter; analysis performed none", []string{"evaluation", "explicit-side-effect-policy"}, first)
	case ".TS", ".EQ", ".PS", ".R1", ".[":
		family := map[string]string{".TS": "tbl", ".EQ": "eqn", ".PS": "pic", ".R1": "refer", ".[": "refer"}[a.Name]
		capture := c
		reason := fmt.Sprintf("%s source captured as data; layout/meaning requires additional interpretation", family)
		if p := n.Parent(); c.active && p != nil && p.Kind() == groff.ASTBlock && p.Attributes().Name == a.Name && !p.Attributes().Closed {
			capture.capability.Syntax = Recognized
			reason += "; source boundary is incomplete"
		}
		return r.emit(capture, "builtin.preprocessor", "preprocessor", family, reason, []string{"preprocess:" + family}, "")
	case ".MR", ".UR", ".MT":
		family := "url"
		if a.Name == ".MR" {
			family = "manual"
			if len(a.Arguments) > 1 {
				first += "(" + a.Arguments[1].Value + ")"
			}
		} else if a.Name == ".MT" {
			family = "mail"
		}
		return emit("reference", "reference", family, "Explicit reference target is data; no target was opened or validated", nil, first)
	}
	return nil
}

func opaqueNode(node *groff.ASTNode) bool {
	line := node
	for line.Parent() != nil && line.Kind() == groff.ASTInline {
		line = line.Parent()
	}
	if node.Attributes().Opaque || line.Attributes().Opaque {
		return true
	}
	for p := line.Parent(); p != nil; p = p.Parent() {
		if p.Kind() == groff.ASTBlock && p.Attributes().Opaque && line.Span().Start.Offset > p.Span().Start.Offset {
			return true
		}
	}
	return false
}
