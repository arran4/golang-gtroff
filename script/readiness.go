package script

import (
	"context"
	"strings"

	"github.com/d5/tengo/v2/parser"
)

// Ready uses the native parser to distinguish a complete cell from a program
// awaiting EOF-delimited syntax. Other syntax failures are ready for Eval, which
// reports the error and charges an attempted cell. Probe bytes are separately
// bounded to limit repeated parsing of a growing multiline program.
func (s *Session) Ready(ctx context.Context, cell Cell) (bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if err := s.ctx.Err(); err != nil {
		return false, err
	}
	if len(cell.Code) > s.o.Cell.MaxScriptBytes || len(cell.Code) > s.o.MaxSessionBytes*16-s.probeBytes {
		return false, ErrLimit
	}
	s.probeBytes += len(cell.Code)
	ctx, cancel := context.WithTimeout(ctx, s.o.Cell.Duration)
	defer cancel()
	file := parser.NewFileSet().AddFile("<input>", -1, len(cell.Code))
	if err := preflightScript(ctx, file, cell.Code, s.o.Cell); err != nil {
		return false, err
	}
	_, err := parser.NewParser(file, cell.Code, nil).ParseFile()
	if err == nil {
		return true, ctx.Err()
	}
	if list, ok := err.(parser.ErrorList); ok {
		incomplete := false
		for _, e := range list {
			if e.Pos.Offset >= len(cell.Code) || strings.Contains(e.Msg, "raw string literal not terminated") || e.Msg == "comment not terminated" {
				incomplete = true
			} else {
				return true, nil
			}
		}
		return !incomplete, nil
	}
	return true, nil
}
