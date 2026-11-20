package script

import (
	"context"
	"fmt"
	"reflect"
	"sort"

	"github.com/d5/tengo/v2"
)

// State copying uses an explicit graph walk. Native Object.Copy recursively
// copies containers and shares closure cells; neither behavior is transactional.
// Retain object, map, closure-cell and overlapping slice backing identities.
// Pointer values are used only as identity/range keys, never dereferenced or
// converted back to pointers. Source objects stay live throughout the copy.
type stateRegion struct {
	start, end uintptr
	objects    []tengo.Object
	bytes      []byte
}
type stateCopier struct {
	checks               int
	maxObjects, maxBytes int
	seen                 map[tengo.Object]tengo.Object
	order                []tengo.Object
	arrays, bytes        []stateRegion
	maps                 map[uintptr]map[string]tengo.Object
	cells                map[*tengo.Object]*tengo.Object
}

func cloneState(ctx context.Context, roots []tengo.Object, o SessionOptions) ([]tengo.Object, error) {
	c := &stateCopier{maxObjects: o.MaxStateObjects, maxBytes: o.MaxStateBytes,
		seen: map[tengo.Object]tengo.Object{}, maps: map[uintptr]map[string]tengo.Object{}, cells: map[*tengo.Object]*tengo.Object{}}
	queue := append([]tengo.Object(nil), roots...)
	objectSize := reflect.TypeOf((*tengo.Object)(nil)).Elem().Size()
	spend := func(nodes, bytes int) error {
		if nodes > c.maxObjects || bytes > c.maxBytes {
			return ErrLimit
		}
		c.maxObjects -= nodes
		c.maxBytes -= bytes
		return nil
	}
	for len(queue) > 0 {
		if err := c.check(ctx); err != nil {
			return nil, err
		}
		value := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		if value == nil {
			continue
		}
		if _, ok := c.seen[value]; ok {
			continue
		}
		if err := spend(1, 64); err != nil {
			return nil, err
		}
		c.seen[value] = nil
		c.order = append(c.order, value)
		var arr []tengo.Object
		var obj map[string]tengo.Object
		switch v := value.(type) {
		case *tengo.Array:
			arr = v.Value
		case *tengo.ImmutableArray:
			arr = v.Value
		case *tengo.Map:
			obj = v.Value
		case *tengo.ImmutableMap:
			obj = v.Value
		case *tengo.Bytes:
			if err := spend(0, cap(v.Value)); err != nil {
				return nil, err
			}
			if cap(v.Value) > 0 {
				a := reflect.ValueOf(v.Value[:cap(v.Value)]).Pointer()
				c.bytes = append(c.bytes, stateRegion{start: a, end: a + uintptr(cap(v.Value))})
			}
		case *tengo.String:
			if err := spend(0, len(v.Value)); err != nil {
				return nil, err
			}
		case *tengo.CompiledFunction:
			if len(v.Free) > c.maxObjects || len(v.Instructions) > c.maxBytes || len(v.SourceMap) > c.maxBytes/16 {
				return nil, ErrLimit
			}
			if err := spend(len(v.Free), len(v.Instructions)+16*len(v.SourceMap)); err != nil {
				return nil, err
			}
			for _, free := range v.Free {
				queue = append(queue, free)
			}
		case *tengo.ObjectPtr:
			if v.Value != nil {
				queue = append(queue, *v.Value)
			}
		case *tengo.Error:
			queue = append(queue, v.Value)
		case *tengo.UserFunction, *tengo.BuiltinFunction, *tengo.Undefined, *tengo.Bool, *tengo.Int, *tengo.Float, *tengo.Char, *tengo.Time:
		default:
			return nil, fmt.Errorf("%w: unsupported retained Tengo object %T", ErrOptions, value)
		}
		if cap(arr) > 0 {
			if cap(arr) > c.maxBytes/16 {
				return nil, ErrLimit
			}
			if err := spend(cap(arr), 16*cap(arr)); err != nil {
				return nil, err
			}
			full := arr[:cap(arr)]
			a := reflect.ValueOf(full).Pointer()
			c.arrays = append(c.arrays, stateRegion{start: a, end: a + uintptr(len(full))*objectSize})
			queue = append(queue, full...)
		}
		if obj != nil {
			if len(obj) > c.maxBytes/32 {
				return nil, ErrLimit
			}
			if err := spend(len(obj), 32*len(obj)); err != nil {
				return nil, err
			}
			for key, item := range obj {
				if err := spend(0, len(key)); err != nil {
					return nil, err
				}
				queue = append(queue, item)
			}
		}
	}
	c.arrays = mergeStateRegions(c.arrays)
	c.bytes = mergeStateRegions(c.bytes)
	for i := range c.arrays {
		r := &c.arrays[i]
		r.objects = make([]tengo.Object, int((r.end-r.start)/objectSize))
	}
	for i := range c.bytes {
		r := &c.bytes[i]
		r.bytes = make([]byte, int(r.end-r.start))
	}
	// Allocate skeletons first so cycles and forward edges resolve in one pass.
	for _, value := range c.order {
		if err := c.check(ctx); err != nil {
			return nil, err
		}
		var next tengo.Object
		switch v := value.(type) {
		case *tengo.Array:
			next = &tengo.Array{Value: c.array(v.Value, objectSize)}
		case *tengo.ImmutableArray:
			next = &tengo.ImmutableArray{Value: c.array(v.Value, objectSize)}
		case *tengo.Bytes:
			var b []byte
			if cap(v.Value) > 0 {
				r := findStateRegion(c.bytes, reflect.ValueOf(v.Value[:cap(v.Value)]).Pointer())
				off := int(reflect.ValueOf(v.Value[:cap(v.Value)]).Pointer() - r.start)
				b = r.bytes[off : off+len(v.Value) : off+cap(v.Value)]
			} else if v.Value != nil {
				b = []byte{}
			}
			next = &tengo.Bytes{Value: b}
		case *tengo.Map:
			next = &tengo.Map{Value: c.objectMap(v.Value)}
		case *tengo.ImmutableMap:
			next = &tengo.ImmutableMap{Value: c.objectMap(v.Value)}
		case *tengo.CompiledFunction:
			next = &tengo.CompiledFunction{Instructions: v.Instructions, NumLocals: v.NumLocals, NumParameters: v.NumParameters, VarArgs: v.VarArgs, SourceMap: v.SourceMap, Free: make([]*tengo.ObjectPtr, len(v.Free))}
		case *tengo.ObjectPtr:
			var target *tengo.Object
			if v.Value != nil {
				target = c.cells[v.Value]
				if target == nil {
					var item tengo.Object
					target = &item
					c.cells[v.Value] = target
				}
			}
			next = &tengo.ObjectPtr{Value: target}
		case *tengo.Error:
			next = &tengo.Error{}
		case *tengo.String:
			next = &tengo.String{Value: v.Value}
		case *tengo.Int:
			next = &tengo.Int{Value: v.Value}
		case *tengo.Float:
			next = &tengo.Float{Value: v.Value}
		case *tengo.Char:
			next = &tengo.Char{Value: v.Value}
		case *tengo.Time:
			next = &tengo.Time{Value: v.Value}
		default:
			// Only session-owned host/built-in functions and singleton values can
			// reach these roots. Native copy(doc) may wrap the same host closure.
			next = value
		}
		c.seen[value] = next
	}
	for _, value := range c.order {
		if err := c.check(ctx); err != nil {
			return nil, err
		}
		var arr, out []tengo.Object
		var obj, dest map[string]tengo.Object
		switch v := value.(type) {
		case *tengo.Array:
			arr = v.Value
			out = c.seen[value].(*tengo.Array).Value
		case *tengo.ImmutableArray:
			arr = v.Value
			out = c.seen[value].(*tengo.ImmutableArray).Value
		case *tengo.Map:
			obj = v.Value
			dest = c.seen[value].(*tengo.Map).Value
		case *tengo.ImmutableMap:
			obj = v.Value
			dest = c.seen[value].(*tengo.ImmutableMap).Value
		case *tengo.Bytes:
			copy(c.seen[value].(*tengo.Bytes).Value[:cap(v.Value)], v.Value[:cap(v.Value)])
		case *tengo.Error:
			c.seen[value].(*tengo.Error).Value = c.seen[v.Value]
		case *tengo.ObjectPtr:
			if v.Value != nil {
				*c.seen[value].(*tengo.ObjectPtr).Value = c.seen[*v.Value]
			}
		case *tengo.CompiledFunction:
			for i, item := range v.Free {
				c.seen[value].(*tengo.CompiledFunction).Free[i] = c.seen[item].(*tengo.ObjectPtr)
			}
		}
		for i, item := range arr[:cap(arr)] {
			out[:cap(out)][i] = c.seen[item]
		}
		for key, item := range obj {
			dest[key] = c.seen[item]
		}
	}
	result := make([]tengo.Object, len(roots))
	for i, item := range roots {
		result[i] = c.seen[item]
	}
	return result, nil
}
func mergeStateRegions(regions []stateRegion) []stateRegion {
	sort.Slice(regions, func(i, j int) bool { return regions[i].start < regions[j].start })
	out := regions[:0]
	for _, r := range regions {
		if len(out) > 0 && r.start < out[len(out)-1].end {
			if r.end > out[len(out)-1].end {
				out[len(out)-1].end = r.end
			}
		} else {
			out = append(out, r)
		}
	}
	return out
}
func findStateRegion(regions []stateRegion, start uintptr) *stateRegion {
	i := sort.Search(len(regions), func(i int) bool { return regions[i].end > start })
	return &regions[i]
}
func (c *stateCopier) array(v []tengo.Object, size uintptr) []tengo.Object {
	if cap(v) == 0 {
		if v == nil {
			return nil
		}
		return []tengo.Object{}
	}
	a := reflect.ValueOf(v[:cap(v)]).Pointer()
	r := findStateRegion(c.arrays, a)
	off := int((a - r.start) / size)
	return r.objects[off : off+len(v) : off+cap(v)]
}
func (c *stateCopier) objectMap(v map[string]tengo.Object) map[string]tengo.Object {
	if v == nil {
		return nil
	}
	id := reflect.ValueOf(v).Pointer()
	m := c.maps[id]
	if m == nil {
		m = make(map[string]tengo.Object, len(v))
		c.maps[id] = m
	}
	return m
}

func (c *stateCopier) check(ctx context.Context) error {
	c.checks++
	return cooperativeContext(ctx, c.checks)
}
