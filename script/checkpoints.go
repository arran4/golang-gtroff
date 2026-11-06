package script

import (
	"context"
	"fmt"
	"reflect"
	goruntime "runtime"
	"strings"
	"time"

	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/parser"
)

const checkpointName = "__gtroff_checkpoint"

// instrument inserts native Tengo calls before statements and in empty blocks.
// This preserves the parsed grammar and original source positions, without
// rewriting bytecode/jump offsets or maintaining a competing language/VM.
// Only parser.Node fields are traversed; source-file metadata is excluded.
func instrument(ctx context.Context, file *parser.File, maxNodes int) error {
	stack := []parser.Node{file}
	for len(stack) > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		maxNodes--
		if maxNodes < 0 {
			return ErrLimit
		}
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		value := reflect.ValueOf(node)
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
				stack = append(stack, child)
				continue
			}
			if field.Kind() == reflect.Slice {
				for j := field.Len() - 1; j >= 0; j-- {
					if child, ok := field.Index(j).Interface().(parser.Node); ok {
						stack = append(stack, child)
					}
				}
			}
		}
		switch n := node.(type) {
		case *parser.File:
			n.Stmts = checkpointStatements(n.Stmts, n.Pos(), false)
		case *parser.BlockStmt:
			n.Stmts = checkpointStatements(n.Stmts, n.LBrace, true)
		}
	}
	return nil
}
func checkpointStatements(statements []parser.Stmt, position parser.Pos, empty bool) []parser.Stmt {
	makeCall := func(pos parser.Pos) parser.Stmt {
		return &parser.ExprStmt{Expr: &parser.CallExpr{Func: &parser.Ident{Name: checkpointName, NamePos: pos}, LParen: pos, RParen: pos}}
	}
	if len(statements) == 0 {
		if empty {
			return []parser.Stmt{makeCall(position)}
		}
		return nil
	}
	result := make([]parser.Stmt, 0, 2*len(statements))
	for _, statement := range statements {
		result = append(result, makeCall(statement.Pos()), statement)
	}
	return result
}
func (r *runtime) checkpoint(args ...tengo.Object) (tengo.Object, error) {
	if len(args) != 0 {
		return nil, tengo.ErrWrongNumArguments
	}
	r.steps++
	if r.steps > r.o.MaxSteps {
		return nil, fmt.Errorf("%w: statement/block checkpoint budget", ErrLimit)
	}
	if err := r.ctx.Err(); err != nil {
		return nil, err
	}
	if r.steps == 1 || r.steps%64 == 0 {
		// Go/js cannot service a timer while an uninterrupted VM occupies the host
		// event loop. Check the deadline clock directly as well as yielding Go tasks.
		if deadline, ok := r.ctx.Deadline(); ok && !time.Now().Before(deadline) {
			return nil, context.DeadlineExceeded
		}
		goruntime.Gosched()
	}
	return tengo.UndefinedValue, nil
}
func reservedIdentifier(name string) bool { return strings.HasPrefix(name, "__gtroff_") }
