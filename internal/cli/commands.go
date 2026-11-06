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
