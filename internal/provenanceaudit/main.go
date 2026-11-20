// provenanceaudit compares catalogue descriptions with explicit local evidence.
// It reports text matches, never copyright ownership or redistribution clearance.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

type options []string

func (o *options) String() string { return strings.Join(*o, ",") }
func (o *options) Set(v string) error {
	label, path, ok := strings.Cut(v, "=")
	if !ok || label == "" || path == "" {
		return fmt.Errorf("expected LABEL=FILE")
	}
	for _, old := range *o {
		if strings.HasPrefix(old, label+"=") {
			return fmt.Errorf("duplicate label %q", label)
		}
	}
	*o = append(*o, v)
	return nil
}

type item struct {
	Type  string
	Token string
	Args  string
}
type group struct {
	Description string
	Contents    []item
}
type evidence struct {
	Label  string `json:"label"`
	SHA256 string `json:"sha256"`
	Kind   string `json:"kind"`
}
type location struct {
	Source string `json:"source"`
	Kind   string `json:"kind"`
	Anchor string `json:"anchor,omitempty"`
	Line   int    `json:"line,omitempty"`
}
type row struct {
	Group           int        `json:"group"`
	DescriptionLine int        `json:"description_line"`
	SHA256          string     `json:"description_sha256"`
	Items           []item     `json:"items"`
	Matches         []location `json:"matches"`
	History         []string   `json:"byte_identical_history"`
	ReflowedHistory []string   `json:"whitespace_identical_history"`
}
type report struct {
	Schema          int            `json:"schema"`
	Revision        string         `json:"revision"`
	CatalogueSHA256 string         `json:"catalogue_sha256"`
	Groups          int            `json:"groups"`
	Items           int            `json:"items"`
	MatchedGroups   int            `json:"matched_groups"`
	UnmatchedGroups int            `json:"unmatched_groups"`
	SourceGroups    map[string]int `json:"source_matched_groups"`
	Sources         []evidence     `json:"sources"`
	Rows            []row          `json:"rows"`
}
type definition struct {
	text, anchor string
	line         int
}
type source struct {
	evidence
	definitions []definition
	text        string
	history     map[string]bool
	reflowed    map[string]bool
}

// Whitespace normalization deliberately retains punctuation, escape spelling,
// Unicode distinctions and prose. A match is not a byte-for-byte identity.
func normalize(s string) string { return strings.Join(strings.Fields(s), " ") }
func digest(b []byte) string    { return fmt.Sprintf("%x", sha256.Sum256(b)) }

func read(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	const limit = 32 << 20
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err == nil && len(b) > limit {
		err = fmt.Errorf("input exceeds 32 MiB: %s", path)
	}
	return b, err
}

func decodeGroups(b []byte) ([]group, error) {
	var groups []group
	if err := json.Unmarshal(b, &groups); err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return nil, fmt.Errorf("empty catalogue")
	}
	for i, g := range groups {
		if normalize(g.Description) == "" || len(g.Contents) == 0 {
			return nil, fmt.Errorf("group %d lacks description or items", i)
		}
		for _, item := range g.Contents {
			if item.Type == "" || item.Token == "" {
				return nil, fmt.Errorf("group %d has an incomplete item", i)
			}
		}
	}
	return groups, nil
}

func textContent(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

func attribute(n *html.Node, name string) string {
	for _, a := range n.Attr {
		if a.Key == name {
			return a.Val
		}
	}
	return ""
}

var definitionHeading = regexp.MustCompile(`(Macro|Request|Register|String):\s+[.\\]`)

func manualDefinitions(raw []byte) ([]definition, error) {
	// Honor the HTML declaration (both examined releases use ISO-8859-1).
	r, err := charset.NewReader(bytes.NewReader(raw), "text/html")
	if err != nil {
		return nil, err
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	// Keep declared anchors and their physical HTML lines, not rendered lines.
	lines := make(map[string]int)
	z, line := html.NewTokenizer(bytes.NewReader(b)), 1
	for {
		kind := z.Next()
		if kind == html.ErrorToken {
			if z.Err() != io.EOF {
				return nil, z.Err()
			}
			break
		}
		tokenLine := line
		line += bytes.Count(z.Raw(), []byte{'\n'})
		if kind == html.StartTagToken {
			t := z.Token()
			for _, a := range t.Attr {
				if a.Key == "id" {
					if _, duplicate := lines[a.Val]; duplicate {
						return nil, fmt.Errorf("duplicate HTML id %q", a.Val)
					}
					lines[a.Val] = tokenLine
				}
			}
		}
	}
	doc, err := html.Parse(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	var result []definition
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "dt" && definitionHeading.MatchString(textContent(n)) {
			// Earlier extractors read nextElementSibling, even another dt. Only
			// a dd proves a definition-body match here. Empty alias dt nodes
			// are not silently associated with an unrelated later description.
			for s := n.NextSibling; s != nil; s = s.NextSibling {
				if s.Type != html.ElementNode {
					continue
				}
				if s.Data == "dd" {
					anchor := attribute(n, "id")
					text := normalize(textContent(s))
					if text != "" {
						result = append(result, definition{text, anchor, lines[anchor]})
					}
				}
				break
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	if len(result) == 0 {
		return nil, fmt.Errorf("no nonempty roff definition bodies found")
	}
	return result, nil
}

func load(spec, kind string) (source, error) {
	label, path, _ := strings.Cut(spec, "=")
	b, err := read(path)
	s := source{evidence: evidence{label, digest(b), kind}}
	if err != nil {
		return s, err
	}
	switch kind {
	case "manual-html":
		s.definitions, err = manualDefinitions(b)
	case "reference-text":
		s.text = normalize(string(b))
		if s.text == "" {
			err = fmt.Errorf("empty reference text")
		}
	case "historical-catalogue":
		var gs []group
		gs, err = decodeGroups(b)
		s.history = make(map[string]bool)
		s.reflowed = make(map[string]bool)
		for _, g := range gs {
			s.history[g.Description] = true
			s.reflowed[normalize(g.Description)] = true
		}
	}
	return s, err
}

func compare(raw []byte, revision string, sources []source) (report, error) {
	groups, err := decodeGroups(raw)
	r := report{Schema: 1, Revision: revision, CatalogueSHA256: digest(raw), Groups: len(groups), SourceGroups: make(map[string]int)}
	if err != nil {
		return r, err
	}
	// JSON description-key positions; a quoted string containing this word is
	// escaped in the source and cannot masquerade as an object property.
	positions := regexp.MustCompile(`"Description"\s*:`).FindAllIndex(raw, -1)
	if len(positions) != len(groups) {
		return r, fmt.Errorf("description property count differs from groups")
	}
	for _, s := range sources {
		r.Sources = append(r.Sources, s.evidence)
		if s.Kind != "historical-catalogue" {
			r.SourceGroups[s.Label] = 0
		}
	}
	for i, g := range groups {
		row := row{Group: i, DescriptionLine: bytes.Count(raw[:positions[i][0]], []byte{'\n'}) + 1, SHA256: digest([]byte(g.Description)), Items: g.Contents, Matches: []location{}, History: []string{}, ReflowedHistory: []string{}}
		r.Items += len(g.Contents)
		text := normalize(g.Description)
		for _, s := range sources {
			matched := false
			for _, d := range s.definitions {
				if text == d.text {
					row.Matches = append(row.Matches, location{s.Label, "definition-whitespace-exact", d.anchor, d.line})
					matched = true
				}
			}
			if s.text != "" && strings.Contains(s.text, text) {
				row.Matches = append(row.Matches, location{Source: s.Label, Kind: "reference-whitespace-substring"})
				matched = true
			}
			if matched {
				r.SourceGroups[s.Label]++
			}
			if s.history[g.Description] {
				row.History = append(row.History, s.Label)
			}
			if s.reflowed[text] {
				row.ReflowedHistory = append(row.ReflowedHistory, s.Label)
			}
		}
		if len(row.Matches) > 0 {
			r.MatchedGroups++
		} else {
			r.UnmatchedGroups++
		}
		r.Rows = append(r.Rows, row)
	}
	return r, nil
}

func run(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("provenanceaudit", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	catalogue := flags.String("catalogue", "cmd/gen/in.json", "catalogue JSON")
	revision := flags.String("revision", "", "reviewed commit (recorded without verification)")
	var manuals, references, history options
	flags.Var(&manuals, "manual", "repeatable LABEL=HTML manual")
	flags.Var(&references, "reference", "repeatable LABEL=TEXT reference")
	flags.Var(&history, "history", "repeatable COMMIT=JSON historical catalogue")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || len(manuals)+len(references) == 0 || *revision == "" {
		return fmt.Errorf("usage: provenanceaudit -revision COMMIT [-catalogue JSON] -manual LABEL=HTML [-reference LABEL=TEXT] [-history COMMIT=JSON]")
	}
	var sources []source
	labels := make(map[string]bool)
	for i, specs := range []options{manuals, references, history} {
		for _, spec := range specs {
			s, err := load(spec, []string{"manual-html", "reference-text", "historical-catalogue"}[i])
			if err != nil {
				return fmt.Errorf("%s: %w", spec, err)
			}
			if labels[s.Label] {
				return fmt.Errorf("duplicate evidence label %q", s.Label)
			}
			labels[s.Label] = true
			sources = append(sources, s)
		}
	}
	b, err := read(*catalogue)
	if err != nil {
		return err
	}
	r, err := compare(b, *revision, sources)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "provenanceaudit:", err)
		os.Exit(1)
	}
}
