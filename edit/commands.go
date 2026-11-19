package edit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/arran4/golang-gtroff/query"
)

// CommandSpec is a bounded, declarative command representation for CLI/hosts.
// It has no loops, imports or expression evaluation. Tengo owns procedural state.
// Source Text is a JSON UTF-8 string; Go PlanPatches supports arbitrary bytes.
type CommandSpec struct {
	Selector    string      `json:"selector"`
	View        query.View  `json:"view,omitempty"`
	Restrict    Restriction `json:"restrict,omitempty"`
	Operation   Operation   `json:"operation"`
	Text        string      `json:"text,omitempty"`
	Find        string      `json:"find,omitempty"`
	Heading     string      `json:"heading,omitempty"`
	Term        string      `json:"term,omitempty"`
	Destination string      `json:"destination,omitempty"`
	Rewrite     bool        `json:"rewrite,omitempty"`
}

func (s CommandSpec) Command() (Command, error) {
	q, err := query.ParseSelector(s.Selector)
	if err != nil {
		return Command{}, err
	}
	c := Command{Select: q, View: s.View, Restrict: s.Restrict, Operation: s.Operation, Text: s.Text, Find: s.Find, Heading: s.Heading, Term: s.Term, Rewrite: s.Rewrite}
	if s.Destination != "" {
		c.Destination, err = query.ParseSelector(s.Destination)
		if err != nil {
			return c, err
		}
	}
	return c, validateCommand(c)
}

// ParseCommands accepts at most 64 KiB, schemaVersion 1 and 256 commands. Unknown
// fields/trailing JSON and invalid selectors fail before any command executes.
func ParseCommands(data []byte) ([]Command, error) {
	if len(data) > 64<<10 {
		return nil, fmt.Errorf("%w: command file exceeds 64 KiB", ErrOptions)
	}
	var file struct {
		SchemaVersion int           `json:"schemaVersion"`
		Commands      []CommandSpec `json:"commands"`
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&file); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOptions, err)
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("%w: trailing command data", ErrOptions)
	}
	if file.SchemaVersion != 1 || len(file.Commands) == 0 || len(file.Commands) > 256 {
		return nil, ErrOptions
	}
	result := make([]Command, 0, len(file.Commands))
	for i, spec := range file.Commands {
		c, err := spec.Command()
		if err != nil {
			return nil, fmt.Errorf("command %d: %w", i+1, err)
		}
		result = append(result, c)
	}
	return result, nil
}
