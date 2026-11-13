package analyze

import (
	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/source"
)

func addProfileEvidence(profile *Profile, file *source.File, node *groff.ASTNode, catalogue []groff.CatalogueDefinition) {
	name := node.Attributes().Name
	kind, reason := "", ""
	switch name {
	case ".TH":
		kind = "man"
		reason = "man-style title header spelling; runtime redefinition is not evaluated"
	case ".TL", ".AU", ".NH":
		kind = "ms"
		reason = "ms-style title/author/numbered-heading spelling"
	case ".Dd", ".Dt", ".Nm", ".Sh":
		kind = "mdoc"
		reason = "mdoc-style spelling; mdoc semantics are not implemented"
	case ".SH", ".TP", ".IP":
		kind = "ambiguous-fragment"
		reason = "common man fragment spelling; heading names can be package-ambiguous"
	case ".de", ".de1", ".dei", ".dei1", ".am", ".am1":
		kind = "custom"
		reason = "macro-definition syntax without executing its body"
	}
	if kind == "" {
		core, foreign := false, false
		for _, entry := range catalogue {
			if entry.Package == "roff" {
				core = true
			} else {
				foreign = true
			}
		}
		if core && !foreign {
			kind = "roff"
			reason = "Core roff control spelling without a package-specific title"
		}
	}
	if kind != "" {
		profile.Evidence = append(profile.Evidence, Evidence{file.ID(), node.ID(), node.Span(), kind, reason})
	}
}
func resolveProfile(evidence []Evidence) Profile {
	observed := map[string]int{}
	for _, e := range evidence {
		observed[e.Profile]++
	}
	p := Profile{Name: "unknown", Confidence: "low", Reason: "no distinctive active macro-package evidence", Evidence: append([]Evidence(nil), evidence...)}
	strong := 0
	for _, name := range []string{"man", "ms", "mdoc"} {
		if observed[name] > 0 {
			strong++
			p.Name = name
		}
	}
	if strong > 1 {
		p.Name = "mixed"
		p.Confidence = "high"
		p.Reason = "source set contains multiple package-specific spelling families"
		return p
	}
	if strong == 1 {
		p.Confidence = "medium"
		p.Reason = "package-specific source spellings, without evaluating redefinitions"
		if p.Name == "man" || observed[p.Name] > 1 {
			p.Confidence = "high"
		}
		return p
	}
	if observed["custom"] > 0 {
		p.Name = "custom"
		p.Confidence = "medium"
		p.Reason = "definition syntax without a distinctive package title"
		return p
	}
	if observed["roff"] > 0 {
		p.Name = "roff"
		p.Confidence = "medium"
		p.Reason = "core roff syntax without a distinctive package title"
	}
	return p
}
