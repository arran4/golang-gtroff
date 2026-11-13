package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/arran4/golang-gtroff/agentskill"
	"github.com/arran4/golang-gtroff/cmd"
)

const skillsHelp = `Usage: gtroff skills COMMAND [OPTIONS]

Discover portable, versioned instructions for query/extraction workflows.
  list    List bundled skills and content manifest identities
  show    Read a bundled SKILL.md or manifest
  export  Export to an explicit local directory, refusing existing destinations

These are static instruction files, not runtime plugins or executable scripts.
No network downloads, agent-home/config discovery, registration or shell execution.
Export is also the explicit local installation mechanism: choose the parent
directory yourself. Existing files/skills, including identical exports, are never
replaced. There is no implicit installation or replacement flag.
Run 'gtroff help skills list', 'gtroff help skills show' or 'gtroff help skills export'.
`
const skillsListHelp = `Usage: gtroff skills list [--format text|json]

List bundled portable skills, their versions and deterministic manifest hashes.
Default: text. JSON schemaVersion 1 contains skills; no filesystem/network access.
`
const skillsShowHelp = `Usage: gtroff skills show [--format text|json] NAME

Read a bundled skill. Text prints SKILL.md; JSON prints its deterministic manifest.
The manifest records schema/capability requirements and hashes of authored files.
No install, file open or execution occurs. Default: text.
`
const skillsExportHelp = `Usage: gtroff skills export --dir DIRECTORY [--format text|json] NAME

Export bundled portable instruction files into DIRECTORY/NAME. The parent is
explicit; no agent config/home directory is guessed. Missing parent directories
are created. Existing destinations (including identical skills, files and symlinks)
are refused without modification. Select a new target to keep local customizations.
All content stages/syncs in an owned sibling before one directory rename. Use
exclusive writer access; this is not protection against hostile filesystem races.
Text/JSON receipts describe the completed local export. Default: text.
No remote downloads, registration, shell, plugins or bundled-code execution.
Exit codes: 0 success, 1 existing-target conflict, 2 usage/name, 3 I/O/cancellation.
`

// Skills is a subcommand `gtroff skills` Discover and export portable query/extraction recipes
//
// Usage: gtroff skills COMMAND [OPTIONS]
//
// Static, versioned instruction files focused on query/extraction and provenance.
// Commands: list, show, export. Export requires an explicit local parent directory;
// no guessed agent paths, remote downloads, registration, execution or replacement.
func Skills() error { return cmd.ErrPrintHelp }

// SkillsList is a subcommand `gtroff skills list` List bundled portable skill versions and hashes
//
// Usage: gtroff skills list [--format text|json]
//
// Read-only discovery. JSON schemaVersion 1 contains deterministic manifest identities.
//
// Flags:
//
//	format: --format (default: "text") Output: text or json
func SkillsList(format string) error { return skillExit(runSkillsList(format, getEnv())) }

// SkillsShow is a subcommand `gtroff skills show` Read a bundled entrypoint or content manifest
//
// Usage: gtroff skills show [--format text|json] NAME
//
// Text prints SKILL.md; JSON prints its deterministic manifest. No file opens or execution.
//
// Flags:
//
//	format: --format (default: "text") Output: text or json
//	names: ... Exactly one bundled skill name
func SkillsShow(format string, names ...string) error {
	return skillExit(runSkillsShow(format, names, getEnv()))
}

// SkillsExport is a subcommand `gtroff skills export` Export portable files into an explicit new local directory
//
// Usage: gtroff skills export --dir DIRECTORY [--format text|json] NAME
//
// Explicit local installation/export to DIRECTORY/NAME, never guessed agent paths.
// Existing destinations are refused, including identical bundles and symlinks.
// Files stage/sync before one directory rename; exclusive writer access is required.
// No registration, network download, runtime plugins, shell or artifact execution.
//
// Flags:
//
//	directory: --dir (default: "") Explicit parent for the new skill directory
//	format: --format (default: "text") Receipt: text or json
//	names: ... Exactly one bundled skill name
func SkillsExport(directory string, format string, names ...string) error {
	return skillExit(runSkillsExport(directory, format, names, getEnv()))
}
func skillExit(code int) error {
	if code != ExitOK {
		return &cmd.ErrExitCode{Code: code}
	}
	return nil
}

func skillsCommand(args []string, e Environment) int {
	if len(args) == 0 {
		return writeString(e, skillsHelp)
	}
	command := args[0]
	args = args[1:]
	if command == "--help" || command == "-h" {
		return writeString(e, skillsHelp)
	}
	format, directory := "text", ""
	f := flag.NewFlagSet("skills "+command, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	f.StringVar(&format, "format", "text", "")
	if command == "export" {
		f.StringVar(&directory, "dir", "", "")
	}
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return skillsLeafHelp(e, command)
		}
		return usage(e, err.Error())
	}
	switch command {
	case "list":
		if f.NArg() != 0 {
			return usage(e, "skills list takes no names")
		}
		return runSkillsList(format, e)
	case "show":
		return runSkillsShow(format, f.Args(), e)
	case "export":
		return runSkillsExport(directory, format, f.Args(), e)
	}
	return usage(e, "unknown skills command")
}
func skillsLeafHelp(e Environment, command string) int {
	switch command {
	case "list":
		return writeString(e, skillsListHelp)
	case "show":
		return writeString(e, skillsShowHelp)
	case "export":
		return writeString(e, skillsExportHelp)
	}
	return usage(e, "unknown skills command")
}
func runSkillsList(format string, e Environment) int {
	if format != "text" && format != "json" {
		return usage(e, "skills format must be text or json")
	}
	skills, err := agentskill.List()
	if err != nil {
		return failure(e, err.Error())
	}
	if format == "json" {
		data, err := json.Marshal(struct {
			SchemaVersion int               `json:"schemaVersion"`
			Skills        []agentskill.Info `json:"skills"`
		}{1, skills})
		if err != nil {
			return failure(e, err.Error())
		}
		return writeSkillBytes(e, append(data, '\n'))
	}
	var out bytes.Buffer
	for _, info := range skills {
		fmt.Fprintf(&out, "%s %s sha256:%s\n", info.Name, info.Version, info.ManifestSHA256)
	}
	return writeSkillBytes(e, out.Bytes())
}
func runSkillsShow(format string, names []string, e Environment) int {
	if len(names) != 1 || format != "text" && format != "json" {
		return usage(e, "skills show expects one name and format text or json")
	}
	bundle, err := agentskill.Lookup(names[0])
	if err != nil {
		return usage(e, err.Error())
	}
	path := "SKILL.md"
	if format == "json" {
		path = "manifest.json"
	}
	data, err := bundle.ReadFile(path)
	if err != nil {
		return failure(e, err.Error())
	}
	return writeSkillBytes(e, data)
}
func runSkillsExport(directory, format string, names []string, e Environment) int {
	if directory == "" || len(names) != 1 || format != "text" && format != "json" {
		return usage(e, "skills export requires --dir, one name and format text or json")
	}
	if _, err := agentskill.Lookup(names[0]); err != nil {
		return usage(e, err.Error())
	}
	if e.ExportSkill == nil {
		return failure(e, "explicit local skill export capability unavailable")
	}
	ctx := e.Context
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return failure(e, err.Error())
	}
	receipt, err := e.ExportSkill(ctx, names[0], directory)
	if err != nil {
		if errors.Is(err, agentskill.ErrConflict) {
			fmt.Fprintln(e.Err, err)
			return ExitInvalid
		}
		return failure(e, err.Error())
	}
	if receipt == nil {
		return failure(e, "invalid skill export receipt")
	}
	if format == "json" {
		data, err := json.Marshal(receipt)
		if err != nil {
			return failure(e, err.Error())
		}
		return writeSkillBytes(e, append(data, '\n'))
	}
	return writeString(e, fmt.Sprintf("Exported %s %s to %s (%d files); existing skills were not replaced.\n", receipt.Info.Name, receipt.Info.Version, receipt.Directory, len(receipt.Files)))
}

func writeSkillBytes(e Environment, data []byte) int {
	if err := writeBytes(e.Out, data); err != nil {
		return failure(e, err.Error())
	}
	return ExitOK
}
