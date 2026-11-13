package script

import (
	"fmt"
	"sort"

	"github.com/arran4/golang-gtroff/edit"
	"github.com/arran4/golang-gtroff/query"
	"github.com/d5/tengo/v2"
)

// Decode the existing typed command shape directly. A JSON round-trip would
// normalize invalid UTF-8 in exact-source strings and would silently turn byte
// values into base64 strings. Native field types must remain explicit instead.
func command(value tengo.Object) (edit.Command, error) {
	fields, err := objectFields(value)
	if err != nil {
		return edit.Command{}, err
	}
	spec := edit.CommandSpec{}
	keys := sortedFields(fields)
	for _, key := range keys {
		value := fields[key]
		if key == "restrict" {
			spec.Restrict, err = restriction(value)
			if err != nil {
				return edit.Command{}, err
			}
			continue
		}
		if key == "rewrite" {
			spec.Rewrite, err = boolArgument(value)
			if err != nil {
				return edit.Command{}, err
			}
			continue
		}
		text, err := stringArgument(value)
		if err != nil {
			return edit.Command{}, fmt.Errorf("command field %s: %w", key, err)
		}
		switch key {
		case "selector":
			spec.Selector = text
		case "operation":
			spec.Operation = edit.Operation(text)
		case "text":
			spec.Text = text
		case "find":
			spec.Find = text
		case "heading":
			spec.Heading = text
		case "term":
			spec.Term = text
		case "destination":
			spec.Destination = text
		case "view":
			spec.View = query.View(text)
		default:
			return edit.Command{}, fmt.Errorf("%w: unknown command field %q", ErrOptions, key)
		}
	}
	return spec.Command()
}
func restriction(value tengo.Object) (edit.Restriction, error) {
	fields, err := objectFields(value)
	if err != nil {
		return edit.Restriction{}, err
	}
	r := edit.Restriction{}
	for _, key := range sortedFields(fields) {
		value := fields[key]
		if key == "all" {
			r.All, err = boolArgument(value)
			if err != nil {
				return r, err
			}
			continue
		}
		n, ok := value.(*tengo.Int)
		if !ok || n.Value < 1 || uint64(n.Value) > uint64(^uint(0)>>1) {
			return r, fmt.Errorf("%w: restriction indices must be positive integers", ErrOptions)
		}
		switch key {
		case "nth":
			r.Nth = int(n.Value)
		case "from":
			r.From = int(n.Value)
		case "to":
			r.To = int(n.Value)
		case "firstN":
			r.FirstN = int(n.Value)
		default:
			return r, fmt.Errorf("%w: unknown restriction %q", ErrOptions, key)
		}
	}
	return r, nil
}
func boolArgument(value tengo.Object) (bool, error) {
	v, ok := value.(*tengo.Bool)
	if !ok {
		return false, fmt.Errorf("%w: expected boolean argument", ErrOptions)
	}
	return !v.IsFalsy(), nil
}
func objectFields(value tengo.Object) (map[string]tengo.Object, error) {
	switch v := value.(type) {
	case *tengo.Map:
		return v.Value, nil
	case *tengo.ImmutableMap:
		return v.Value, nil
	}
	return nil, fmt.Errorf("%w: expected command data map", ErrOptions)
}
func sortedFields(fields map[string]tengo.Object) []string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
