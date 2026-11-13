package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/arran4/golang-gtroff/cmd"
)

const commandsHelp = `Usage: gtroff commands [--format json|text]

Write a machine-readable inventory of commands, flags, descriptions and capabilities to stdout.
Default format is json.

  --format json|text      Output format, default: json

Exit codes: 0 success, 2 usage, 3 input/output failure.
`

type CommandInventoryItem struct {
	Name         string        `json:"name"`
	Synopsis     string        `json:"synopsis"`
	Description  string        `json:"description"`
	Flags        []CommandFlag `json:"flags,omitempty"`
	Capabilities []string      `json:"capabilities,omitempty"`
}

type CommandFlag struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description"`
}

type CommandsResult struct {
	SchemaVersion int                    `json:"schemaVersion"`
	Command       string                 `json:"command"`
	Commands      []CommandInventoryItem `json:"commands"`
}

// Commands is a subcommand `gtroff commands` Write a machine-readable command inventory to stdout
//
// Usage: gtroff commands [--format json|text]
//
// Write a machine-readable inventory of commands, flags, descriptions and capabilities to stdout.
// Default format is json.
//
// Flags:
//
//	format: --format (default: "json") Output format: json or text
func Commands(format string) error {
	code := runCommands(format, getEnv())
	if code != ExitOK {
		return &cmd.ErrExitCode{Code: code}
	}
	return nil
}

func GetInventory() []CommandInventoryItem {
	return []CommandInventoryItem{
		{
			Name:        "validate",
			Synopsis:    "gtroff validate [--strict] [--package man|ms|roff] [--format text|json] [FILE]",
			Description: "Check roff syntax and report diagnostics",
			Flags: []CommandFlag{
				{Name: "strict", Type: "bool", Default: "false", Description: "Reject warnings, including unknown controls and unloaded includes"},
				{Name: "package", Type: "string", Default: "man", Description: "Package to use: man, ms or roff"},
				{Name: "format", Type: "string", Default: "text", Description: "Output format: text or json"},
			},
			Capabilities: []string{"syntax-inspection", "diagnostics", "json-output", "text-output"},
		},
		{
			Name:        "ast",
			Synopsis:    "gtroff ast [--strict] [--package man|ms|roff] [--format json|text] [--select SELECTOR] [FILE]",
			Description: "Inspect a source-aware syntax tree",
			Flags: []CommandFlag{
				{Name: "strict", Type: "bool", Default: "false", Description: "Reject warnings, including unknown controls and unloaded includes"},
				{Name: "package", Type: "string", Default: "man", Description: "Package to use: man, ms or roff"},
				{Name: "format", Type: "string", Default: "json", Description: "Output format: json or text"},
				{Name: "select", Type: "string", Default: "", Description: "Select syntax nodes: node:ID, path:/0/1, section:HEADING"},
			},
			Capabilities: []string{"ast-export", "node-selection", "json-output", "text-output"},
		},
		{
			Name: "sources", Synopsis: "gtroff sources [OPTIONS] [FILE]", Description: "Inspect bounded local includes and source provenance",
			Flags: []CommandFlag{
				{Name: "no-includes", Type: "bool", Default: "false", Description: "Disable include loading, retaining references"},
				{Name: "root", Type: "string", Description: "Explicit allowed source root"},
				{Name: "strict", Type: "bool", Default: "false", Description: "Reject syntax/include diagnostics"},
				{Name: "package", Type: "string", Default: "man", Description: "Package: man, ms or roff"},
				{Name: "format", Type: "string", Default: "json", Description: "Output format: json or text"},
				{Name: "max-bytes", Type: "int", Default: "8388608", Description: "Cumulative captured source bytes"},
				{Name: "max-nodes", Type: "int", Default: "100000", Description: "Cumulative AST node budget"},
				{Name: "max-files", Type: "int", Default: "256", Description: "Source occurrence cap"},
				{Name: "max-depth", Type: "int", Default: "16", Description: "Include depth, ceiling 128"},
			}, Capabilities: []string{"source-graph", "bounded-includes", "original-source-provenance", "json-output", "text-output"},
		},
		{Name: "query", Synopsis: "gtroff query [OPTIONS] SELECTOR [FILE]", Description: "Select source-aware documentation with stateless structural queries",
			Flags: []CommandFlag{
				{Name: "view", Type: "string", Default: "semantic", Description: "Semantic man structure or physical syntax"},
				{Name: "format", Type: "string", Default: "json", Description: "json, jsonl, text, raw, grep, csv or tsv"},
				{Name: "fields", Type: "string", Description: "Comma-separated CSV/TSV fields"},
				{Name: "no-includes", Type: "bool", Default: "false", Description: "Disable include loading, retaining references"},
				{Name: "root", Type: "string", Description: "Explicit allowed source root"},
				{Name: "strict", Type: "bool", Default: "false", Description: "Reject syntax/include diagnostics"},
				{Name: "package", Type: "string", Default: "man", Description: "Package: man, ms or roff"},
				{Name: "max-bytes", Type: "int", Default: "8388608", Description: "Cumulative input bytes"},
				{Name: "max-nodes", Type: "int", Default: "100000", Description: "Cumulative input AST nodes"},
				{Name: "max-files", Type: "int", Default: "256", Description: "Source occurrence cap"},
				{Name: "max-depth", Type: "int", Default: "16", Description: "Include depth, ceiling 128"},
				{Name: "max-results", Type: "int", Default: "10000", Description: "Maximum query matches"},
				{Name: "max-work", Type: "int", Default: "67108864", Description: "Query work budget"},
				{Name: "max-result-bytes", Type: "int", Default: "16777216", Description: "Materialized match-data budget"},
			}, Capabilities: []string{"stateless-query", "structural-selectors", "syntax-view", "semantic-projection", "source-provenance", "bounded-includes", "json-output", "jsonl-output", "text-output", "raw-output", "grep-output", "csv-output", "tsv-output"}},
		{Name: "extract", Synopsis: "gtroff extract [OPTIONS] SELECTOR [FILE]", Description: "Export selected source fragments and structured documentation records", Flags: []CommandFlag{
			{Name: "view", Type: "string", Default: "semantic", Description: "Select semantic man structure or syntax"},
			{Name: "format", Type: "string", Default: "jsonl", Description: "json, jsonl, text, raw, grep, csv, tsv or html"},
			{Name: "fields", Type: "string", Default: "", Description: "Comma-separated CSV/TSV fields"},
			{Name: "normalize", Type: "bool", Default: "false", Description: "Fold whitespace in text output, preserving case"},
			{Name: "o", Type: "string", Default: "-", Description: "Explicit atomic output file or '-' for stdout"},
			{Name: "strict", Type: "bool", Default: "false", Description: "Reject source and extraction diagnostics"},
			{Name: "root", Type: "string", Default: "", Description: "Explicit allowed source root"},
			{Name: "no-includes", Type: "bool", Default: "false", Description: "Disable include loading, retaining references"},
			{Name: "package", Type: "string", Default: "man", Description: "Parser context: man, ms or roff"},
			{Name: "max-bytes", Type: "int", Default: "8388608", Description: "Cumulative input bytes"},
			{Name: "max-nodes", Type: "int", Default: "100000", Description: "Cumulative input AST nodes"},
			{Name: "max-files", Type: "int", Default: "256", Description: "Source occurrence cap"},
			{Name: "max-depth", Type: "int", Default: "16", Description: "Include depth, ceiling 128"},
			{Name: "max-extract-nodes", Type: "int", Default: "200000", Description: "Projection/construction node budget"},
			{Name: "max-records", Type: "int", Default: "10000", Description: "Maximum records before a resource error"},
			{Name: "max-record-bytes", Type: "int", Default: "16777216", Description: "Materialized record-data budget"},
			{Name: "max-work", Type: "int", Default: "67108864", Description: "Shared query/construction work budget"},
			{Name: "max-output-bytes", Type: "int", Default: "33554432", Description: "Serialized output byte budget"},
		}, Capabilities: []string{"record-extraction", "stateless-query", "definition-records", "source-provenance", "bounded-includes", "explicit-atomic-output", "contributed-exporters", "json-output", "jsonl-output", "text-output", "raw-output", "grep-output", "csv-output", "tsv-output", "html-output"}},
		{Name: "edit", Synopsis: "gtroff edit [OPTIONS] [FILE]", Description: "Plan source-backed transformations and explicit guarded writes", Flags: []CommandFlag{
			{Name: "select", Type: "string", Default: "source[source-id=1]", Description: "Stateless selector for each command"},
			{Name: "operation", Type: "string", Default: "", Description: "Transformation operation"},
			{Name: "text", Type: "string", Default: "", Description: "Literal data or exact source bytes, depending on operation"},
			{Name: "find", Type: "string", Default: "", Description: "Nonempty exact source bytes to replace"},
			{Name: "heading", Type: "string", Default: "", Description: "Literal heading for append-section"},
			{Name: "term", Type: "string", Default: "", Description: "Literal term for append-definition"},
			{Name: "destination", Type: "string", Default: "", Description: "Unique move destination selector"},
			{Name: "view", Type: "string", Default: "semantic", Description: "Select semantic man structure or syntax"},
			{Name: "nth", Type: "int", Default: "-1", Description: "One-based selected match; -1 means unset"},
			{Name: "range", Type: "string", Default: "", Description: "Inclusive one-based match range, e.g. 2:5"},
			{Name: "first", Type: "int", Default: "-1", Description: "First N matches; -1 means unset"},
			{Name: "all", Type: "bool", Default: "false", Description: "Explicitly select all matches"},
			{Name: "rewrite", Type: "bool", Default: "false", Description: "Permit canonical replacement of complex description source"},
			{Name: "commands", Type: "string", Default: "", Description: "Explicit schema-v1 JSON command file, maximum 64 KiB"},
			{Name: "create", Type: "bool", Default: "false", Description: "Create a new in-memory document without reading an input"},
			{Name: "title", Type: "string", Default: "", Description: "Literal title for a new document"},
			{Name: "section", Type: "string", Default: "", Description: "Manual section for a new document"},
			{Name: "format", Type: "string", Default: "diff", Description: "Review diff, JSON or exact raw source on stdout"},
			{Name: "source", Type: "string", Default: "", Description: "Explicit captured physical source for raw/copy output"},
			{Name: "o", Type: "string", Default: "-", Description: "Explicit transformed source copy destination"},
			{Name: "in-place", Type: "bool", Default: "false", Description: "Explicit guarded replacement of changed original files"},
			{Name: "dry-run", Type: "bool", Default: "false", Description: "Preview without filesystem writes, even with output flags"},
			{Name: "include-edits", Type: "bool", Default: "false", Description: "Explicitly allow selected included-file edits"},
			{Name: "strict", Type: "bool", Default: "false", Description: "Require one match and reject source diagnostics"},
			{Name: "no-includes", Type: "bool", Default: "false", Description: "Disable include loading, retaining references"},
			{Name: "root", Type: "string", Default: "", Description: "Explicit allowed source root"},
			{Name: "package", Type: "string", Default: "man", Description: "Parser context: man, ms or roff"},
			{Name: "max-bytes", Type: "int", Default: "8388608", Description: "Cumulative source bytes and working-file budget"},
			{Name: "max-nodes", Type: "int", Default: "100000", Description: "Cumulative parser node budget"},
			{Name: "max-files", Type: "int", Default: "256", Description: "Source occurrence cap"},
			{Name: "max-depth", Type: "int", Default: "16", Description: "Include depth, ceiling 128"},
			{Name: "max-patches", Type: "int", Default: "10000", Description: "Maximum patches and query matches"},
			{Name: "max-work", Type: "int", Default: "268435456", Description: "Cumulative query, byte-processing and reparse work"},
			{Name: "max-output-bytes", Type: "int", Default: "33554432", Description: "Serialized preview/output byte cap"},
		}, Capabilities: []string{"source-backed-editing", "stateless-query", "selection-restrictions", "in-memory-plans", "explicit-guarded-output", "bounded-includes", "json-output", "diff-output", "raw-output"}},
		{Name: "script", Synopsis: "gtroff script [OPTIONS] SCRIPT [DOCUMENT]", Description: "Execute optional Tengo with explicit document capabilities", Flags: []CommandFlag{
			{Name: "repl", Type: "bool", Default: "false", Description: "Persistent native cells from stdin"},
			{Name: "prompt", Type: "bool", Default: "true", Description: "REPL prompts on stderr"},
			{Name: "max-cells", Type: "int", Default: "1000", Description: "Cumulative attempted REPL cells"},
			{Name: "max-session-bytes", Type: "int", Default: "4194304", Description: "Total REPL input bytes"},
			{Name: "allow-edits", Type: "bool", Default: "false", Description: "Grant in-memory editing; never grants filesystem writes"},
			{Name: "include-edits", Type: "bool", Default: "false", Description: "Allow selected included-file edits with allow-edits"},
			{Name: "write-target", Type: "string", Default: "", Description: "Authorize one explicit destination for doc.write requests"},
			{Name: "in-place", Type: "bool", Default: "false", Description: "Explicitly commit changed named original files after success"},
			{Name: "dry-run", Type: "bool", Default: "false", Description: "Suppress filesystem output, including authorized requests"},
			{Name: "strict", Type: "bool", Default: "false", Description: "Reject source diagnostics and ambiguous edit selections"},
			{Name: "no-includes", Type: "bool", Default: "false", Description: "Disable include loading, retaining references"},
			{Name: "root", Type: "string", Default: "", Description: "Explicit allowed source root for the document"},
			{Name: "package", Type: "string", Default: "man", Description: "Parser context: man, ms or roff"},
			{Name: "max-bytes", Type: "int", Default: "8388608", Description: "Cumulative input and working document bytes"},
			{Name: "max-nodes", Type: "int", Default: "100000", Description: "Cumulative source AST nodes"},
			{Name: "max-files", Type: "int", Default: "256", Description: "Source occurrence cap"},
			{Name: "max-depth", Type: "int", Default: "16", Description: "Include depth, ceiling 128"},
			{Name: "max-script-bytes", Type: "int", Default: "65536", Description: "Script source byte cap"},
			{Name: "max-host-calls", Type: "int", Default: "1000", Description: "Maximum host method calls"},
			{Name: "max-steps", Type: "int", Default: "100000", Description: "Statement/block checkpoints, not VM instructions"},
			{Name: "max-allocs", Type: "int", Default: "100000", Description: "Tengo object allocation count, not heap bytes"},
			{Name: "timeout-ms", Type: "int", Default: "1000", Description: "Compile and execution deadline in milliseconds"},
			{Name: "max-output-bytes", Type: "int", Default: "16777216", Description: "Staged emissions and write-request bytes"},
		}, Capabilities: []string{"optional-tengo", "persistent-session", "transactional-cells", "pending-publication-retry", "stateless-query", "in-memory-transforms", "staged-output", "explicit-guarded-output", "bounded-includes"}},
		{Name: "analyze", Synopsis: "gtroff analyze [OPTIONS] [FILE]", Description: "Classify syntax, profiles and processing requirements without execution",
			Flags: []CommandFlag{
				{Name: "format", Type: "string", Default: "text", Description: "text, json or jsonl"},
				{Name: "details", Type: "bool", Default: "false", Description: "Detailed evidence in text output"},
				{Name: "rules", Type: "string", Description: "Explicit declarative JSON rules file"},
				{Name: "no-includes", Type: "bool", Default: "false", Description: "Disable include loading, retaining references"},
				{Name: "root", Type: "string", Description: "Explicit allowed source root"},
				{Name: "strict", Type: "bool", Default: "false", Description: "Reject syntax/include diagnostics"},
				{Name: "package", Type: "string", Default: "man", Description: "Parser context: man, ms or roff"},
				{Name: "max-bytes", Type: "int", Default: "8388608", Description: "Cumulative input bytes"},
				{Name: "max-nodes", Type: "int", Default: "100000", Description: "Cumulative input AST nodes"},
				{Name: "max-files", Type: "int", Default: "256", Description: "Source occurrence cap"},
				{Name: "max-depth", Type: "int", Default: "16", Description: "Include depth, ceiling 128"},
				{Name: "max-analysis-nodes", Type: "int", Default: "200000", Description: "Analysis enumeration node budget"},
				{Name: "max-findings", Type: "int", Default: "100000", Description: "Maximum analysis findings"},
				{Name: "max-work", Type: "int", Default: "67108864", Description: "Node/rule/source-byte work budget"},
			}, Capabilities: []string{"feature-analysis", "profile-evidence", "source-provenance", "bounded-includes", "declarative-rules", "json-output", "jsonl-output", "text-output"}},
		{Name: "lint", Synopsis: "gtroff lint [OPTIONS] [FILE ...]", Description: "Apply structural man documentation policy to explicit source snapshots", Flags: []CommandFlag{
			{Name: "format", Type: "string", Default: "text", Description: "Output: text, json or jsonl"},
			{Name: "rules", Type: "string", Default: "", Description: "Explicit JSON policy file"},
			{Name: "enable", Type: "string", Default: "", Description: "Comma-separated rule IDs to enable"},
			{Name: "disable", Type: "string", Default: "", Description: "Comma-separated rule IDs to disable"},
			{Name: "required", Type: "string", Default: "", Description: "Comma-separated required section headings"},
			{Name: "fail-on", Type: "string", Default: "auto", Description: "Threshold: auto, error, warning, info or never"},
			{Name: "strict", Type: "bool", Default: "false", Description: "Stricter policy and all source diagnostics"},
			{Name: "closed-world", Type: "bool", Default: "false", Description: "Check manual references within a complete collection"},
			{Name: "no-includes", Type: "bool", Default: "false", Description: "Disable include loading, retaining references"},
			{Name: "root", Type: "string", Default: "", Description: "Explicit allowed source root"},
			{Name: "package", Type: "string", Default: "man", Description: "Parser context: man, ms or roff"},
			{Name: "max-bytes", Type: "int", Default: "8388608", Description: "Cumulative collection captured bytes"},
			{Name: "max-nodes", Type: "int", Default: "100000", Description: "Cumulative collection input AST nodes"},
			{Name: "max-files", Type: "int", Default: "256", Description: "Cumulative source occurrence cap"},
			{Name: "max-depth", Type: "int", Default: "16", Description: "Include depth, ceiling 128"},
			{Name: "max-documents", Type: "int", Default: "256", Description: "Root document cap"},
			{Name: "max-lint-nodes", Type: "int", Default: "200000", Description: "Cumulative analysis AST node budget"},
			{Name: "max-findings", Type: "int", Default: "100000", Description: "Maximum findings including suppressed"},
			{Name: "max-work", Type: "int", Default: "67108864", Description: "Shared charged work budget"},
		}, Capabilities: []string{"structural-lint", "extensible-rules", "policy-suppression", "collection-checks", "source-provenance", "bounded-includes", "json-output", "jsonl-output", "text-output"}},
		{Name: "diff", Synopsis: "gtroff diff [OPTIONS] BEFORE AFTER", Description: "Compare documentation revisions with original-source provenance", Flags: []CommandFlag{
			{Name: "view", Type: "string", Default: "semantic", Description: "Compare semantic man content or source bytes"},
			{Name: "format", Type: "string", Default: "text", Description: "Output: text, json or jsonl"},
			{Name: "strict", Type: "bool", Default: "false", Description: "Reject source/model/ambiguity diagnostics"},
			{Name: "root", Type: "string", Default: "", Description: "Shared explicit allowed source root"},
			{Name: "no-includes", Type: "bool", Default: "false", Description: "Compare explicit root sources without loading includes"},
			{Name: "package", Type: "string", Default: "man", Description: "Parser context: man, ms or roff"},
			{Name: "max-bytes", Type: "int", Default: "8388608", Description: "Input bytes across both roots"},
			{Name: "max-nodes", Type: "int", Default: "100000", Description: "Input AST nodes across both roots"},
			{Name: "max-files", Type: "int", Default: "256", Description: "Source occurrences across both roots"},
			{Name: "max-depth", Type: "int", Default: "16", Description: "Include depth, ceiling 128"},
			{Name: "max-diff-nodes", Type: "int", Default: "200000", Description: "Per-revision projection node bound"},
			{Name: "max-objects", Type: "int", Default: "10000", Description: "Per-revision object cap"},
			{Name: "max-changes", Type: "int", Default: "10000", Description: "Changes plus ambiguities cap"},
			{Name: "max-data-bytes", Type: "int", Default: "33554432", Description: "Comparison data budget"},
			{Name: "max-result-bytes", Type: "int", Default: "16777216", Description: "Materialized change data budget"},
			{Name: "max-work", Type: "int", Default: "67108864", Description: "Shared comparison work budget"},
			{Name: "max-output-bytes", Type: "int", Default: "33554432", Description: "Serialized output byte budget"},
		}, Capabilities: []string{"semantic-diff", "source-diff", "change-provenance", "ambiguous-matches", "bounded-includes", "json-output", "jsonl-output", "text-output"}},
		{
			Name:        "render",
			Synopsis:    "gtroff render --format html|png|jpg [OPTIONS] [FILE]",
			Description: "Export a semantic man page to HTML or paginated images",
			Flags: []CommandFlag{
				{Name: "format", Type: "string", Default: "html", Description: "Output format: html, png or jpg"},
				{Name: "o", Type: "string", Default: "-", Description: "Output destination; '-' means stdout"},
				{Name: "section", Type: "string", Description: "Select all matching sections/subsections"},
				{Name: "node", Type: "string", Description: "Select an AST source ID shown by gtroff ast"},
				{Name: "theme", Type: "string", Default: "light", Description: "Color theme: light or dark"},
				{Name: "title", Type: "string", Description: "Override the document title"},
				{Name: "fragment", Type: "bool", Default: "false", Description: "Write styled article without document shell"},
				{Name: "no-anchors", Type: "bool", Default: "false", Description: "Omit heading permalink anchors"},
				{Name: "strict", Type: "bool", Default: "false", Description: "Reject any semantic/render diagnostic"},
				{Name: "assets", Type: "string", Default: "embed", Description: "Font asset policy: embed or omit-fonts"},
				{Name: "font-size", Type: "float64", Default: "18", Description: "Font size in pixels (HTML) or points (images)"},
				{Name: "line-height", Type: "float64", Default: "1.5", Description: "Line height ratio: 1..3"},
				{Name: "max-width", Type: "int", Default: "960", Description: "Max article width in pixels"},
				{Name: "font-regular", Type: "string", Description: "Explicit custom TTF/OTF font file (alias: --font)"},
				{Name: "font-bold", Type: "string", Description: "Explicit custom bold font file"},
				{Name: "font-italic", Type: "string", Description: "Explicit custom italic font file"},
				{Name: "font-mono", Type: "string", Description: "Explicit custom mono font file"},
				{Name: "width", Type: "int", Default: "1000", Description: "Image page width in pixels"},
				{Name: "margin", Type: "int", Default: "48", Description: "Image page margin in pixels"},
				{Name: "dpi", Type: "float64", Default: "96", Description: "Image resolution in pixels/inch"},
				{Name: "page-height", Type: "int", Default: "1400", Description: "Image page height in pixels"},
				{Name: "single", Type: "bool", Default: "false", Description: "Derive one image's height within guarded limits"},
				{Name: "max-height", Type: "int", Default: "32768", Description: "Image max height in pixels"},
				{Name: "max-pixels", Type: "int64", Default: "67108864", Description: "Total pixel ceiling across pages"},
				{Name: "quality", Type: "int", Default: "90", Description: "JPEG encoding quality: 1..100"},
			},
			Capabilities: []string{"html-export", "png-export", "jpeg-export", "pagination", "custom-fonts"},
		},
		{
			Name:        "image generate",
			Synopsis:    "gtroff image generate [--format png|jpg] [OPTIONS] [FILE]",
			Description: "Render a man page to paginated PNG or JPEG images",
			Flags: []CommandFlag{
				{Name: "format", Type: "string", Default: "png", Description: "Output format: png or jpg"},
				{Name: "o", Type: "string", Default: "-", Description: "Output image file; '-' means stdout"},
				{Name: "theme", Type: "string", Default: "light", Description: "Color theme: light or dark"},
				{Name: "title", Type: "string", Description: "Override the document title"},
				{Name: "section", Type: "string", Description: "Select all matching sections/subsections"},
				{Name: "node", Type: "string", Description: "Select an AST source ID shown by gtroff ast"},
				{Name: "strict", Type: "bool", Default: "false", Description: "Reject any semantic/render diagnostic"},
				{Name: "width", Type: "int", Default: "1000", Description: "Image page width in pixels"},
				{Name: "margin", Type: "int", Default: "48", Description: "Image page margin in pixels"},
				{Name: "font-size", Type: "float64", Default: "15", Description: "Font size in points: 6..72"},
				{Name: "dpi", Type: "float64", Default: "96", Description: "Image resolution in pixels/inch"},
				{Name: "line-height", Type: "float64", Default: "1.35", Description: "Line height ratio: 1..3"},
				{Name: "page-height", Type: "int", Default: "1400", Description: "Image page height in pixels"},
				{Name: "single", Type: "bool", Default: "false", Description: "Derive one image height within guarded limits"},
				{Name: "max-height", Type: "int", Default: "32768", Description: "Image max height in pixels"},
				{Name: "max-pixels", Type: "int64", Default: "67108864", Description: "Total pixel ceiling across pages"},
				{Name: "quality", Type: "int", Default: "90", Description: "JPEG encoding quality: 1..100"},
			},
			Capabilities: []string{"png-export", "jpeg-export", "pagination", "custom-fonts"},
		},
		{
			Name:        "convert",
			Synopsis:    "gtroff convert --from html|roff --to roff|html [-i FILE] [-o FILE] [--strict]",
			Description: "Convert supported HTML and man structure",
			Flags: []CommandFlag{
				{Name: "from", Type: "string", Description: "Choose html or roff"},
				{Name: "to", Type: "string", Description: "Choose roff or html"},
				{Name: "i", Type: "string", Default: "-", Description: "Input file; '-' means stdin"},
				{Name: "o", Type: "string", Default: "-", Description: "Output destination; '-' means stdout"},
				{Name: "strict", Type: "bool", Default: "false", Description: "Reject any conversion diagnostic"},
				{Name: "fragment", Type: "bool", Default: "false", Description: "Emit styled article for HTML"},
				{Name: "assets", Type: "string", Default: "embed", Description: "Font asset policy for HTML: embed or omit-fonts"},
			},
			Capabilities: []string{"html-to-roff", "roff-to-html", "semantic-normalization"},
		},
		{
			Name:        "serve",
			Synopsis:    "gtroff serve --root DIR [--root DIR ...] [OPTIONS]",
			Description: "Browse and search explicit man-page directories over HTTP",
			Flags: []CommandFlag{
				{Name: "root", Type: "[]string", Description: "Required, repeatable explicit man directory"},
				{Name: "listen", Type: "string", Default: "127.0.0.1:8080", Description: "TCP address to listen on"},
				{Name: "system-search", Type: "bool", Default: "false", Description: "Probe apropos/whatis fallback"},
				{Name: "allow-remote", Type: "bool", Default: "false", Description: "Permit non-loopback binding"},
				{Name: "max-pages", Type: "int", Default: "10000", Description: "Index cap: 1..50000"},
				{Name: "max-total-bytes", Type: "int64", Default: "67108864", Description: "Source/decode byte ceiling"},
				{Name: "max-file-bytes", Type: "int64", Default: "2097152", Description: "Raw/decoded per-page cap"},
				{Name: "theme", Type: "string", Default: "light", Description: "Page theme: light or dark"},
				{Name: "assets", Type: "string", Default: "embed", Description: "Font asset policy"},
			},
			Capabilities: []string{"http-server", "manlib-index", "read-only-sandbox"},
		},
		{
			Name:        "view",
			Synopsis:    "gtroff view [OPTIONS] FILE",
			Description: "Open a man source file in the optional Shiny viewer",
			Flags: []CommandFlag{
				{Name: "title", Type: "string", Description: "Optional window/document title override"},
				{Name: "section", Type: "string", Description: "Display matching semantic sections"},
				{Name: "width", Type: "int", Default: "1000", Description: "Render page width in pixels"},
				{Name: "page-height", Type: "int", Default: "1400", Description: "Render page height in pixels"},
				{Name: "margin", Type: "int", Default: "48", Description: "Page margin in pixels"},
				{Name: "font-size", Type: "float64", Default: "15", Description: "Default 15 at --dpi 96"},
				{Name: "dpi", Type: "float64", Default: "96", Description: "Document pixels/inch"},
				{Name: "theme", Type: "string", Default: "light", Description: "Color theme: light or dark"},
				{Name: "strict", Type: "bool", Default: "false", Description: "Reject conversion diagnostics"},
				{Name: "window-width", Type: "int", Default: "1000", Description: "Initial viewport width"},
				{Name: "window-height", Type: "int", Default: "800", Description: "Initial viewport height"},
			},
			Capabilities: []string{"desktop-viewer-launcher", "process-forwarding"},
		},
		{
			Name:        "commands",
			Synopsis:    "gtroff commands [--format json|text]",
			Description: "Write a machine-readable command inventory to stdout",
			Flags: []CommandFlag{
				{Name: "format", Type: "string", Default: "json", Description: "Output format: json or text"},
			},
			Capabilities: []string{"command-inventory", "agent-discovery", "json-schema"},
		},
		{
			Name:         "man",
			Synopsis:     "gtroff man [COMMAND]",
			Description:  "Write a deterministic man(7) page for gtroff or one command to stdout",
			Capabilities: []string{"man-page-generation", "deterministic-output"},
		},
		{
			Name:         "completion",
			Synopsis:     "gtroff completion bash|zsh|fish|powershell",
			Description:  "Write shell completion source to stdout",
			Capabilities: []string{"shell-completions", "bash", "zsh", "fish", "powershell"},
		},
		{
			Name:         "version",
			Synopsis:     "gtroff version",
			Description:  "Show build information",
			Capabilities: []string{"version-metadata"},
		},
		{
			Name:         "help",
			Synopsis:     "gtroff help [COMMAND]",
			Description:  "Show help for a command",
			Capabilities: []string{"interactive-help"},
		},
	}
}

func runCommands(format string, e Environment) int {
	if format != "json" && format != "text" {
		return usage(e, "format must be json or text")
	}
	inventory := GetInventory()
	if format == "json" {
		r := CommandsResult{
			SchemaVersion: 1,
			Command:       "commands",
			Commands:      inventory,
		}
		data, err := json.MarshalIndent(r, "", "  ")
		if err != nil {
			return failure(e, "encode commands JSON: "+err.Error())
		}
		data = append(data, '\n')
		return writeString(e, string(data))
	}

	var sb strings.Builder
	sb.WriteString("COMMANDS:\n")
	for _, cmd := range inventory {
		fmt.Fprintf(&sb, "  %-16s %s\n", cmd.Name, cmd.Description)
	}
	return writeString(e, sb.String())
}

func commandsCommand(args []string, e Environment) int {
	format := "json"
	flags := flag.NewFlagSet("commands", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&format, "format", "json", "")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return writeString(e, commandsHelp)
		}
		return usage(e, err.Error())
	}
	if flags.NArg() != 0 {
		return usage(e, "commands takes no positional arguments")
	}
	return runCommands(format, e)
}
