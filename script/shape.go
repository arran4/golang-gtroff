package script

import (
	"context"
	"fmt"
	"reflect"

	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/parser"
)

// Tengo v2.17 encodes call counts, local indices and closure counts in one byte.
// Reject unsupported shapes before truncated operands can execute. The distinct
// identifier bound is conservative across a function and its nested functions;
// it also bounds captured variables forwarded through intermediate closures.
func nativeShape(ctx context.Context, file *parser.File, work int) error {
	type scope struct {
		parent *scope
		names  map[string]bool
	}
	type entry struct {
		node  parser.Node
		scope *scope
	}
	stack := []entry{{node: file}}
	for len(stack) > 0 {
		if err := cooperativeContext(ctx, work); err != nil {
			return err
		}
		work--
		if work < 0 {
			return ErrLimit
		}
		next := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if _, ok := next.node.(*parser.FuncLit); ok {
			next.scope = &scope{parent: next.scope, names: map[string]bool{}}
		}
		if call, ok := next.node.(*parser.CallExpr); ok && len(call.Args) > 255 {
			return fmt.Errorf("%w: Tengo call has more than 255 explicit arguments", ErrLimit)
		}
		if name, ok := next.node.(*parser.Ident); ok {
			for s := next.scope; s != nil; s = s.parent {
				work--
				if work < 0 {
					return ErrLimit
				}
				s.names[name.Name] = true
				if len(s.names) > 255 {
					return fmt.Errorf("%w: Tengo function identifier/capture bound 255", ErrLimit)
				}
			}
		}
		value := reflect.ValueOf(next.node)
		if value.Kind() != reflect.Pointer || value.IsNil() {
			continue
		}
		value = value.Elem()
		for i := 0; i < value.NumField(); i++ {
			field := value.Field(i)
			if !field.CanInterface() {
				continue
			}
			if child, ok := field.Interface().(parser.Node); ok {
				stack = append(stack, entry{child, next.scope})
				continue
			}
			if field.Kind() == reflect.Slice {
				for j := field.Len() - 1; j >= 0; j-- {
					if child, ok := field.Index(j).Interface().(parser.Node); ok {
						stack = append(stack, entry{child, next.scope})
					}
				}
			}
		}
	}
	return nil
}
func nativeConstantBounds(code *tengo.Bytecode) error {
	if len(code.Constants) > 65535 {
		return ErrLimit
	}
	for _, value := range code.Constants {
		if f, ok := value.(*tengo.CompiledFunction); ok && (f.NumLocals > 256 || f.NumParameters > 255) {
			return fmt.Errorf("%w: Tengo function local/parameter operand bound", ErrLimit)
		}
	}
	return nil
}
