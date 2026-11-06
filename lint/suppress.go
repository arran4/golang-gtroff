package lint

import (
	"fmt"
	"sort"
	"strings"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/source"
)

type suppression struct {
	source               source.ID
	rule                 string
	start, end, nextLine int
	reason               string
}

func parseSuppressions(c *Context, registry *Registry, meter *budget) ([]suppression, []ConfigDiagnostic, error) {
	known := map[string]bool{}
	var all []string
	for _, rule := range registry.rules {
		known[rule.Info.ID] = true
		all = append(all, rule.Info.ID)
	}
	sort.Strings(all)
	var out []suppression
	var diagnostics []ConfigDiagnostic
	for _, file := range c.document.Sources.Files() {
		active := map[string]int{}
		var walkErr error
		file.AST().Root().Walk(func(node *groff.ASTNode) groff.WalkAction {
			if err := meter.spend(1); err != nil {
				walkErr = err
				return groff.WalkStop
			}
			if node.Kind() != groff.ASTLine || node.Attributes().TokenKind != groff.KindComment {
				return groff.WalkContinue
			}
			for p := node; p != nil; p = p.Parent() {
				if p.Attributes().Opaque {
					return groff.WalkPrune
				}
			}
			raw := node.Raw()
			if !strings.HasPrefix(raw, `.\"`) && !strings.HasPrefix(raw, `'\"`) {
				return groff.WalkPrune
			}
			text := strings.TrimSpace(raw[3:])
			if !strings.HasPrefix(text, "gtroff-lint-") {
				return groff.WalkPrune
			}
			if err := meter.spend(len(text)); err != nil {
				walkErr = err
				return groff.WalkStop
			}
			parts := strings.Fields(text)
			valid := len(parts) == 2
			command := ""
			if len(parts) > 0 {
				command = strings.TrimPrefix(parts[0], "gtroff-lint-")
			}
			if command != "disable" && command != "enable" && command != "disable-next-line" {
				valid = false
			}
			var ids []string
			if valid {
				ids = strings.Split(parts[1], ",")
				if parts[1] == "all" {
					ids = all
				}
				for _, id := range ids {
					if !known[id] {
						valid = false
					}
				}
			}
			if !valid {
				diagnostics = append(diagnostics, ConfigDiagnostic{c.index, file.ID(), node.Span(), "invalid lint suppression; expected gtroff-lint-disable, enable or disable-next-line followed by known comma-separated rule IDs (or all)"})
				return groff.WalkPrune
			}
			for _, id := range ids {
				if err := meter.spend(1); err != nil {
					walkErr = err
					return groff.WalkStop
				}
				switch command {
				case "disable":
					if _, ok := active[id]; !ok {
						active[id] = len(out)
						out = append(out, suppression{file.ID(), id, node.Span().Start.Offset, len(file.AST().Source()) + 1, 0, fmt.Sprintf("%s:%d: %s", file.Name(), node.Span().Start.Line, parts[0])})
					}
				case "enable":
					if index, ok := active[id]; ok {
						out[index].end = node.Span().Start.Offset
						delete(active, id)
					}
				case "disable-next-line":
					out = append(out, suppression{file.ID(), id, 0, 0, node.Span().Start.Line + 1, fmt.Sprintf("%s:%d: %s", file.Name(), node.Span().Start.Line, parts[0])})
				}
			}
			return groff.WalkPrune
		})
		if walkErr != nil {
			return out, diagnostics, walkErr
		}
	}
	return out, diagnostics, nil
}
