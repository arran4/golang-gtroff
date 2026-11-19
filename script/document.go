package script

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"github.com/arran4/golang-gtroff/edit"
	"github.com/arran4/golang-gtroff/extract"
	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/query"
	"github.com/d5/tengo/v2"
)

func (r *runtime) documentObject() tengo.Object {
	functions := map[string]tengo.CallableFunc{
		"preview": r.preview, "quote": r.quote, "query": r.query, "extract": r.extract, "edit": r.edit, "create": r.create,
		"source": r.source, "sources": r.sources, "emit": r.emit, "emit_json": r.emitJSON, "write": r.write,
	}
	values := map[string]tengo.Object{}
	for name, call := range functions {
		values[name] = r.function("doc."+name, call)
	}
	return &tengo.ImmutableMap{Value: values}
}
func stringArgument(value tengo.Object) (string, error) {
	s, ok := value.(*tengo.String)
	if !ok {
		return "", fmt.Errorf("%w: expected string argument", ErrOptions)
	}
	return s.Value, nil
}
func (r *runtime) requireDocument() error {
	if r.working == nil {
		return ErrDocument
	}
	return nil
}
func (r *runtime) query(args ...tengo.Object) (tengo.Object, error) {
	if err := r.requireDocument(); err != nil {
		return nil, err
	}
	if len(args) < 1 || len(args) > 2 {
		return nil, tengo.ErrWrongNumArguments
	}
	text, err := stringArgument(args[0])
	if err != nil {
		return nil, err
	}
	selector, err := query.ParseSelector(text)
	if err != nil {
		return nil, err
	}
	options := query.Options{}
	if len(args) == 2 {
		view, err := stringArgument(args[1])
		if err != nil {
			return nil, err
		}
		options.View = query.View(view)
	}
	result, err := r.working.Query(r.ctx, selector, options)
	if err != nil {
		return nil, err
	}
	matches := make([]tengo.Object, 0, len(result.Matches))
	for _, m := range result.Matches {
		matches = append(matches, &tengo.ImmutableMap{Value: map[string]tengo.Object{
			"index": &tengo.Int{Value: int64(m.Index)}, "kind": &tengo.String{Value: string(m.Kind)}, "name": &tengo.String{Value: m.Name}, "term": &tengo.String{Value: m.Term}, "text": &tengo.String{Value: m.Text}, "raw": &tengo.Bytes{Value: []byte(m.Raw)},
			"source": &tengo.String{Value: m.Source.Name}, "digest": &tengo.String{Value: m.Source.Digest}, "line": &tengo.Int{Value: int64(m.Span.Start.Line)}, "column": &tengo.Int{Value: int64(m.Span.Start.Column)}, "offset": &tengo.Int{Value: int64(m.Span.Start.Offset)}, "end": &tengo.Int{Value: int64(m.Span.End.Offset)},
		}})
	}
	return &tengo.ImmutableArray{Value: matches}, nil
}
func (r *runtime) extract(args ...tengo.Object) (tengo.Object, error) {
	if err := r.requireDocument(); err != nil {
		return nil, err
	}
	if len(args) != 2 {
		return nil, tengo.ErrWrongNumArguments
	}
	text, err := stringArgument(args[0])
	if err != nil {
		return nil, err
	}
	format, err := stringArgument(args[1])
	if err != nil {
		return nil, err
	}
	selector, err := query.ParseSelector(text)
	if err != nil {
		return nil, err
	}
	f := extract.Format(format)
	options := extract.Options{Strict: r.o.Edit.Strict, HTML: f == extract.HTML}
	writeOptions := extract.WriteOptions{MaxBytes: r.o.MaxOutputBytes}
	if err = extract.ValidateFormat(f, writeOptions); err != nil {
		return nil, err
	}
	report, err := extract.Select(r.ctx, r.working.Sources(), selector, options)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err = report.Write(r.ctx, &out, f, writeOptions); err != nil {
		return nil, err
	}
	return &tengo.Bytes{Value: append([]byte(nil), out.Bytes()...)}, nil
}
func (r *runtime) edit(args ...tengo.Object) (tengo.Object, error) {
	if !r.o.AllowEdits {
		return nil, ErrDenied
	}
	if err := r.requireDocument(); err != nil {
		return nil, err
	}
	if len(args) != 1 {
		return nil, tengo.ErrWrongNumArguments
	}
	command, err := command(args[0])
	if err != nil {
		return nil, err
	}
	plan, err := r.working.Plan(r.ctx, command)
	if err != nil {
		return nil, err
	}
	r.diagnostics = append(r.diagnostics, plan.Diagnostics()...)
	if err = r.working.Apply(plan); err != nil {
		return nil, err
	}
	return &tengo.Int{Value: int64(len(plan.Patches()))}, nil
}
func (r *runtime) create(args ...tengo.Object) (tengo.Object, error) {
	if !r.o.AllowEdits {
		return nil, ErrDenied
	}
	if len(args) != 1 && len(args) != 3 {
		return nil, tengo.ErrWrongNumArguments
	}
	if r.working != nil && (r.created || len(r.working.Changes()) > 0) {
		return nil, fmt.Errorf("%w: create would discard pending edits", edit.ErrTarget)
	}
	name, err := stringArgument(args[0])
	if err != nil {
		return nil, err
	}
	metadata := man.Metadata{}
	if len(args) == 3 {
		metadata.Title, err = stringArgument(args[1])
		if err != nil {
			return nil, err
		}
		metadata.Section, err = stringArgument(args[2])
		if err != nil {
			return nil, err
		}
	}
	w, err := edit.Create(r.ctx, name, metadata, r.o.Edit)
	if err != nil {
		return nil, err
	}
	r.working = w
	r.created = true
	return tengo.UndefinedValue, nil
}
func (r *runtime) source(args ...tengo.Object) (tengo.Object, error) {
	if err := r.requireDocument(); err != nil {
		return nil, err
	}
	if len(args) > 1 {
		return nil, tengo.ErrWrongNumArguments
	}
	name := r.working.RootName()
	if len(args) == 1 {
		var err error
		name, err = stringArgument(args[0])
		if err != nil {
			return nil, err
		}
	}
	b, err := r.working.Data(name)
	if err != nil {
		return nil, err
	}
	if len(b) > r.o.MaxOutputBytes {
		return nil, ErrLimit
	}
	return &tengo.Bytes{Value: b}, nil
}
func (r *runtime) sources(args ...tengo.Object) (tengo.Object, error) {
	if err := r.requireDocument(); err != nil {
		return nil, err
	}
	if len(args) != 0 {
		return nil, tengo.ErrWrongNumArguments
	}
	var names []tengo.Object
	for _, name := range r.working.Names() {
		names = append(names, &tengo.String{Value: name})
	}
	return &tengo.ImmutableArray{Value: names}, nil
}
func (r *runtime) emit(args ...tengo.Object) (tengo.Object, error) {
	if len(args) != 1 {
		return nil, tengo.ErrWrongNumArguments
	}
	var data []byte
	switch value := args[0].(type) {
	case *tengo.String:
		data = []byte(value.Value)
	case *tengo.Bytes:
		data = value.Value
	default:
		return nil, fmt.Errorf("%w: emit accepts string/bytes; use emit_json for records", ErrOptions)
	}
	if len(data) > r.o.MaxOutputBytes-len(r.output) {
		return nil, ErrLimit
	}
	r.output = append(r.output, data...)
	return tengo.UndefinedValue, nil
}
func (r *runtime) emitJSON(args ...tengo.Object) (tengo.Object, error) {
	if len(args) != 1 {
		return nil, tengo.ErrWrongNumArguments
	}
	data, err := r.jsonValue(args[0])
	if err != nil {
		return nil, err
	}
	if len(data) >= r.o.MaxOutputBytes-len(r.output) {
		return nil, ErrLimit
	}
	r.output = append(r.output, data...)
	r.output = append(r.output, '\n')
	return tengo.UndefinedValue, nil
}
func (r *runtime) write(args ...tengo.Object) (tengo.Object, error) {
	if !r.o.AllowWriteRequests {
		return nil, ErrDenied
	}
	if err := r.requireDocument(); err != nil {
		return nil, err
	}
	if len(args) != 1 && len(args) != 2 {
		return nil, tengo.ErrWrongNumArguments
	}
	name := r.working.RootName()
	arg := 0
	if len(args) == 2 {
		var err error
		name, err = stringArgument(args[0])
		if err != nil {
			return nil, err
		}
		arg = 1
	}
	if len(r.writes) > 0 && (r.working != r.writeWorking || r.working.Version() != r.writeVersion) {
		return nil, edit.ErrStale
	}
	destination, err := stringArgument(args[arg])
	if err != nil {
		return nil, err
	}
	if destination == "" || destination == "-" {
		return nil, fmt.Errorf("%w: write requires a named explicit destination", ErrOptions)
	}
	for _, w := range r.writes {
		if w.Destination == destination {
			return nil, fmt.Errorf("%w: duplicate write destination", edit.ErrOverlap)
		}
	}
	data, err := r.working.Data(name)
	if err != nil {
		return nil, err
	}
	size := len(data)
	for _, w := range r.writes {
		size += len(w.Data)
	}
	if size > r.o.MaxOutputBytes {
		return nil, ErrLimit
	}
	r.writes = append(r.writes, WriteRequest{Source: name, Destination: destination, Data: data})
	r.writeVersion = r.working.Version()
	r.writeWorking = r.working
	return tengo.UndefinedValue, nil
}

// Convert only supported data types, never invoking Object.String (which may
// recurse through cyclic script data). Node/depth/byte limits precede encoding.
func (r *runtime) jsonValue(value tengo.Object) ([]byte, error) {
	remaining := r.o.MaxOutputBytes
	nodes := 100000
	active := map[tengo.Object]bool{}
	var convert func(tengo.Object, int) (any, error)
	convert = func(value tengo.Object, depth int) (any, error) {
		if err := r.ctx.Err(); err != nil {
			return nil, err
		}
		nodes--
		if depth > 32 || nodes < 0 {
			return nil, ErrLimit
		}
		spend := func(n int) error {
			if n > remaining {
				return ErrLimit
			}
			remaining -= n
			return nil
		}
		if err := spend(16); err != nil {
			return nil, err
		}
		switch v := value.(type) {
		case *tengo.Undefined:
			return nil, nil
		case *tengo.Bool:
			return !v.IsFalsy(), nil
		case *tengo.Int:
			return v.Value, nil
		case *tengo.Float:
			return v.Value, nil
		case *tengo.Char:
			return string(v.Value), nil
		case *tengo.String:
			if len(v.Value) > remaining/6 {
				return nil, ErrLimit
			}
			if err := spend(len(v.Value) * 6); err != nil {
				return nil, err
			}
			return v.Value, nil
		case *tengo.Bytes:
			if len(v.Value) > remaining/2 {
				return nil, ErrLimit
			}
			if err := spend(len(v.Value) * 2); err != nil {
				return nil, err
			}
			return append([]byte(nil), v.Value...), nil
		}
		var array []tengo.Object
		var object map[string]tengo.Object
		switch v := value.(type) {
		case *tengo.Array:
			array = v.Value
		case *tengo.ImmutableArray:
			array = v.Value
		case *tengo.Map:
			object = v.Value
		case *tengo.ImmutableMap:
			object = v.Value
		default:
			return nil, fmt.Errorf("%w: JSON requires data values", ErrOptions)
		}
		if active[value] {
			return nil, fmt.Errorf("%w: cyclic JSON value", ErrOptions)
		}
		active[value] = true
		defer delete(active, value)
		if object != nil {
			if len(object) > nodes {
				return nil, ErrLimit
			}
			out := make(map[string]any, len(object))
			keys := make([]string, 0, len(object))
			for key := range object {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				item := object[key]
				if len(key) > remaining/6 {
					return nil, ErrLimit
				}
				if err := spend(6 * len(key)); err != nil {
					return nil, err
				}
				v, err := convert(item, depth+1)
				if err != nil {
					return nil, err
				}
				out[key] = v
			}
			return out, nil
		}
		if len(array) > nodes {
			return nil, ErrLimit
		}
		out := make([]any, len(array))
		for i, item := range array {
			v, err := convert(item, depth+1)
			if err != nil {
				return nil, err
			}
			out[i] = v
		}
		return out, nil
	}
	converted, err := convert(value, 0)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(converted)
	if len(data) > r.o.MaxOutputBytes {
		return nil, ErrLimit
	}
	return data, err
}

func (r *runtime) quote(args ...tengo.Object) (tengo.Object, error) {
	if len(args) != 1 {
		return nil, tengo.ErrWrongNumArguments
	}
	s, err := stringArgument(args[0])
	if err != nil {
		return nil, err
	}
	if len(s) > r.o.MaxOutputBytes/6 {
		return nil, ErrLimit
	}
	return &tengo.String{Value: strconv.Quote(s)}, nil
}

func (r *runtime) preview(args ...tengo.Object) (tengo.Object, error) {
	if err := r.requireDocument(); err != nil {
		return nil, err
	}
	if len(args) != 1 {
		return nil, tengo.ErrWrongNumArguments
	}
	format, err := stringArgument(args[0])
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err = r.working.WritePreview(r.ctx, &out, format, r.o.MaxOutputBytes); err != nil {
		return nil, err
	}
	return &tengo.Bytes{Value: append([]byte(nil), out.Bytes()...)}, nil
}
