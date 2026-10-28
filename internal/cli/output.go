package cli

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"unicode/utf8"

	groff "github.com/arran4/golang-gtroff"
)

const schemaVersion = 1
const maxExportBytes = 32 << 20

type validationResult struct {
	SchemaVersion int                `json:"schemaVersion"`
	Command       string             `json:"command"`
	SourceName    string             `json:"sourceName"`
	Assessment    string             `json:"assessment"`
	Status        string             `json:"status"`
	Diagnostics   []groff.Diagnostic `json:"diagnostics"`
	Error         string             `json:"error,omitempty"`
}

func result(command, source, status string, diagnostics []groff.Diagnostic, parseErr error) validationResult {
	if diagnostics == nil {
		diagnostics = []groff.Diagnostic{}
	}
	r := validationResult{SchemaVersion: schemaVersion, Command: command, SourceName: source, Assessment: "syntax-only", Status: status, Diagnostics: diagnostics}
	if parseErr != nil {
		r.Error = parseErr.Error()
	}
	return r
}
func outputValidation(e Environment, source string, o inspectOptions, diagnostics []groff.Diagnostic, status string, code int, parseErr error) int {
	r := result("validate", source, status, diagnostics, parseErr)
	if o.format == "json" {
		if err := encodeJSON(e.Out, r); err != nil {
			return failure(e, "write validation: "+err.Error())
		}
		return code
	}
	if err := writeBytes(e.Out, []byte(fmt.Sprintf("%s: %s syntax (semantic validation not performed)\n", source, status))); err != nil {
		return failure(e, "write stdout: "+err.Error())
	}
	if err := writeDiagnostics(e.Err, diagnostics); err != nil {
		return ExitIO
	}
	if parseErr != nil {
		if _, err := fmt.Fprintf(e.Err, "gtroff: %v\n", parseErr); err != nil {
			return ExitIO
		}
	}
	return code
}

type exportedNode struct {
	ID         groff.NodeID        `json:"id"`
	Parent     groff.NodeID        `json:"parent,omitempty"`
	Children   []groff.NodeID      `json:"children,omitempty"`
	Path       string              `json:"path"`
	Kind       groff.ASTKind       `json:"kind"`
	Span       groff.Span          `json:"span"`
	Attributes groff.ASTAttributes `json:"attributes"`
	Raw        string              `json:"raw"`
	// rawBase64/sourceBase64 appear only for invalid UTF-8. Offsets always index
	// original bytes, not encoding/json's UTF-8 replacement representation.
	RawBase64 string             `json:"rawBase64,omitempty"`
	Token     json.RawMessage    `json:"token,omitempty"`
	Inline    *groff.InlineToken `json:"inline,omitempty"`
}
type astResult struct {
	validationResult
	Selector          string         `json:"selector"`
	RootIDs           []groff.NodeID `json:"rootIds"`
	ExternalParentIDs []groff.NodeID `json:"externalParentIds"`
	Source            string         `json:"source"`
	SourceBase64      string         `json:"sourceBase64,omitempty"`
	Nodes             []exportedNode `json:"nodes"`
}

func outputAST(e Environment, source string, o inspectOptions, tree *groff.AST, roots []*groff.ASTNode, diagnostics []groff.Diagnostic, status string, code int, parseErr error) int {
	r := astResult{validationResult: result("ast", source, status, diagnostics, parseErr), Selector: o.selector, Source: tree.Source(), RootIDs: []groff.NodeID{}, ExternalParentIDs: []groff.NodeID{}, Nodes: []exportedNode{}}
	if !utf8.ValidString(r.Source) {
		r.SourceBase64 = base64.StdEncoding.EncodeToString([]byte(r.Source))
	}
	seen := map[groff.NodeID]bool{}
	rawBytes := len(r.Source)
	for _, root := range roots {
		r.RootIDs = append(r.RootIDs, root.ID())
		var exportErr error
		root.Walk(func(n *groff.ASTNode) groff.WalkAction {
			if seen[n.ID()] {
				return groff.WalkPrune
			}
			seen[n.ID()] = true
			rawBytes += len(n.Raw())
			if rawBytes > maxExportBytes {
				exportErr = fmt.Errorf("AST export exceeds %d byte budget", maxExportBytes)
				return groff.WalkStop
			}
			row := exportedNode{ID: n.ID(), Path: n.Path(), Kind: n.Kind(), Span: n.Span(), Attributes: n.Attributes(), Raw: n.Raw()}
			if !utf8.ValidString(row.Raw) {
				row.RawBase64 = base64.StdEncoding.EncodeToString([]byte(row.Raw))
			}
			if p := n.Parent(); p != nil {
				row.Parent = p.ID()
			}
			for _, child := range n.Children() {
				row.Children = append(row.Children, child.ID())
			}
			token, err := n.Token()
			if err != nil {
				exportErr = err
				return groff.WalkStop
			}
			if token != nil {
				row.Token, err = json.Marshal(token)
				if err != nil {
					exportErr = err
					return groff.WalkStop
				}
			}
			if in, ok := n.Inline(); ok {
				row.Inline = &in
			}
			r.Nodes = append(r.Nodes, row)
			return groff.WalkContinue
		})
		if exportErr != nil {
			return failure(e, exportErr.Error())
		}
	}
	// Multiple selected roots can be nested; node rows still appear exactly once
	// in original document order and retain links to any external parent.
	sort.Slice(r.Nodes, func(i, j int) bool { return r.Nodes[i].ID < r.Nodes[j].ID })
	outside := map[groff.NodeID]bool{}
	for _, n := range r.Nodes {
		if n.Parent != 0 && !seen[n.Parent] && !outside[n.Parent] {
			r.ExternalParentIDs = append(r.ExternalParentIDs, n.Parent)
			outside[n.Parent] = true
		}
	}
	if o.format == "json" {
		if err := encodeJSON(e.Out, r); err != nil {
			return failure(e, "write AST: "+err.Error())
		}
		return code
	}
	var b limitedBuffer
	b.limit = maxExportBytes
	for _, n := range r.Nodes {
		name := n.Attributes.Name
		if n.Attributes.Heading != "" {
			name = n.Attributes.Heading
		}
		if _, err := fmt.Fprintf(&b, "%d %s %s %q %s:%d:%d [%d,%d) raw=%q\n", n.ID, n.Path, n.Kind, name, source, n.Span.Start.Line, n.Span.Start.Column, n.Span.Start.Offset, n.Span.End.Offset, n.Raw); err != nil {
			return failure(e, err.Error())
		}
	}
	if err := writeBytes(e.Out, b.Bytes()); err != nil {
		return failure(e, "write stdout: "+err.Error())
	}
	if err := writeDiagnostics(e.Err, diagnostics); err != nil {
		return ExitIO
	}
	if parseErr != nil {
		if _, err := fmt.Fprintf(e.Err, "gtroff: %v\n", parseErr); err != nil {
			return ExitIO
		}
	}
	return code
}

// Buffer only bounded output. A budget/encoding failure leaves stdout empty;
// a failure in the supplied writer can still leave a partial stream.
type limitedBuffer struct {
	bytes.Buffer
	limit int
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if len(p) > b.limit-b.Len() {
		return 0, fmt.Errorf("output exceeds %d byte budget", b.limit)
	}
	return b.Buffer.Write(p)
}
func encodeJSON(w io.Writer, v interface{}) error {
	b := limitedBuffer{limit: maxExportBytes}
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return err
	}
	return writeBytes(w, b.Bytes())
}
func writeBytes(w io.Writer, p []byte) error {
	n, err := w.Write(p)
	if err == nil && n != len(p) {
		return io.ErrShortWrite
	}
	return err
}
func writeDiagnostics(w io.Writer, ds []groff.Diagnostic) error {
	for _, d := range ds {
		if err := writeBytes(w, []byte(fmt.Sprintf("%s:%d:%d: %s: %s\n", d.Position.Filename, d.Position.Line, d.Position.Column, d.Severity, d.Message))); err != nil {
			return err
		}
	}
	return nil
}
func sortDiagnostics(ds []groff.Diagnostic) {
	sort.SliceStable(ds, func(i, j int) bool { return ds[i].Position.Offset < ds[j].Position.Offset })
}
