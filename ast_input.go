package groff

import (
	"errors"
	"fmt"
	"io"
)

// ASTInputError locates an input failure at the end of validated source bytes.
// Unwrap retains original reader errors and an ASTSourceLimitError, even when
// both occur in the same read. Syntax diagnostics remain separate tree data.
type ASTInputError struct {
	Position Position
	Err      error
}

func (e *ASTInputError) Error() string {
	return fmt.Sprintf("%s:%d:%d: %v", e.Position.Filename, e.Position.Line, e.Position.Column, e.Err)
}
func (e *ASTInputError) Unwrap() error { return e.Err }

// ASTSourceLimitError identifies the configured source-byte limit. The extra
// detection byte remains in Source; this error does not imply observed EOF.
type ASTSourceLimitError struct{ Limit int }

func (e *ASTSourceLimitError) Error() string {
	return fmt.Sprintf("AST source budget exceeded (%d bytes)", e.Limit)
}

func readASTInput(reader io.Reader, options ParserOptions) ([]byte, bool, error) {
	limit := int64(options.ASTBytes)
	if limit < 1<<63-1 {
		limit++
	}
	checked := &checkedSourceReader{reader: reader}
	data, err := io.ReadAll(io.LimitReader(checked, limit))
	complete := checked.terminal == io.EOF
	if len(data) > options.ASTBytes {
		err = errors.Join(&ASTSourceLimitError{Limit: options.ASTBytes}, err)
	}
	if err != nil {
		err = &ASTInputError{Position: advanceSourcePosition(Position{Filename: options.SourceName, Line: 1, Column: 1}, string(data)), Err: err}
	}
	return data, complete, err
}
