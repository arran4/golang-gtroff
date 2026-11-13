package docdiff

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/query"
	"github.com/arran4/golang-gtroff/source"
)

type inspected struct {
	input    Input
	info     InputInfo
	keys     map[source.ID]string
	disabled bool
	children bool
}

func inspect(input Input, side string, o Options, m *meter) (*inspected, error) {
	s := &inspected{input: input, info: InputInfo{Name: input.Name, Files: []query.SourceInfo{}}, keys: map[source.ID]string{}}
	if input.Error != nil {
		s.info.Error = input.Error.Error()
	}
	if input.Sources == nil {
		return s, errors.Join(fmt.Errorf("%w: %s has no source snapshot", ErrInput, side), input.Error)
	}
	s.info.Diagnostics = input.Sources.Diagnostics()
	s.info.Includes = input.Sources.Edges()
	edges := map[source.EdgeID]source.Edge{}
	ordinals := map[source.EdgeID]int{}
	counts := map[string]int{}
	var inputErr error
	for _, edge := range s.info.Includes {
		if err := m.spend(len(edge.Target) + 1); err != nil {
			return s, err
		}
		edges[edge.ID] = edge
		key := tuple(strconv.FormatUint(uint64(edge.From), 10), edge.Request, path.Clean(edge.Target))
		counts[key]++
		ordinals[edge.ID] = counts[key]
		if edge.State == "disabled" {
			s.disabled = true
		} else if edge.State != "resolved" {
			inputErr = errors.Join(inputErr, fmt.Errorf("%w: %s include %q is %s", ErrInput, side, edge.Target, edge.State))
		}
	}
	nodes := 0
	for _, file := range input.Sources.Files() {
		if err := m.spend(len(file.AST().Source()) + len(file.Name()) + 1); err != nil {
			return s, err
		}
		var walkErr error
		file.AST().Root().Walk(func(*groff.ASTNode) groff.WalkAction {
			if nodes >= o.MaxNodes {
				walkErr = ErrLimit
				return groff.WalkStop
			}
			nodes++
			if err := m.spend(1); err != nil {
				walkErr = err
				return groff.WalkStop
			}
			return groff.WalkContinue
		})
		if walkErr != nil {
			return s, walkErr
		}
		digest := sha256.Sum256([]byte(file.AST().Source()))
		info := query.SourceInfo{ID: file.ID(), Name: file.Name(), ParentEdge: file.ParentEdge(), Depth: file.Depth(), Digest: fmt.Sprintf("%x", digest), Complete: file.AST().SourceComplete()}
		s.info.Files = append(s.info.Files, info)
		key := "$root"
		if file.ParentEdge() != 0 {
			edge, ok := edges[file.ParentEdge()]
			parent, exists := s.keys[edge.From]
			if !ok || !exists {
				return s, fmt.Errorf("%w: invalid include occurrence graph", ErrInput)
			}
			key = parent + "/" + url.PathEscape(edge.Request) + ":" + url.PathEscape(path.Clean(edge.Target)) + "#" + strconv.Itoa(ordinals[edge.ID])
			s.children = true
		}
		if err := m.spend(len(key)); err != nil {
			return s, err
		}
		s.keys[file.ID()] = key
		if !info.Complete {
			inputErr = errors.Join(inputErr, fmt.Errorf("%w: %s source %q did not reach EOF", ErrInput, side, file.Name()))
		}
	}
	if len(s.info.Files) == 0 {
		inputErr = errors.Join(inputErr, fmt.Errorf("%w: %s has no captured files", ErrInput, side))
	}
	if input.Error != nil && !errors.Is(input.Error, source.ErrStrict) {
		inputErr = errors.Join(inputErr, ErrInput, input.Error)
	}
	if o.View == Semantic {
		for _, d := range s.info.Diagnostics {
			if d.Severity == "error" {
				inputErr = errors.Join(inputErr, fmt.Errorf("%w: %s has syntax errors", ErrInput, side))
				break
			}
		}
	}
	s.info.Complete = inputErr == nil
	return s, inputErr
}
func tuple(parts ...string) string {
	var out strings.Builder
	for _, part := range parts {
		out.WriteString(strconv.Itoa(len(part)))
		out.WriteByte(':')
		out.WriteString(part)
	}
	return out.String()
}
