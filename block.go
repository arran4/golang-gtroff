package groff

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// BlockLine retains a physical line (or opaque recovery tail), including its
// ending and byte span.
type BlockLine struct {
	RawSource string
	Span      Span
}

func (l BlockLine) Text() string { return documentLineBody(l.RawSource) }

// BlockHeader supplies syntax only. Command keeps its original . or ' prefix;
// Args excludes leading argument separators. Line retains the exact header.
type BlockHeader struct {
	Command, Args, Package string
	Line                   BlockLine
}

// BlockPair declares a literal opener and its alternative closing commands.
// Commands use dot-normalized spelling, including the leading dot.
type BlockPair struct {
	Start string
	Ends  []string
}

// BlockRule controls parser-owned capture. Nested pairs are recognized only
// inside this block; their contents remain raw and never dispatch handlers.
// With no Nested pairs, the first matching terminator closes the block.
type BlockRule struct {
	Ends   []string
	Nested []BlockPair
}

// BlockSource is a bounded, non-evaluated block snapshot. Body excludes the
// header and outer terminator, but includes any nested delimiters. Closed is a
// source fact; a closed block can still contain a mismatched delimiter.
type BlockSource struct {
	Header     BlockHeader
	Rule       BlockRule
	Body       []BlockLine
	Tail       BlockLine // Buffered but unparsed bytes after an input/resource failure.
	Terminator BlockLine
	Closed     bool
}

func (b BlockSource) Source() string {
	var out strings.Builder
	out.WriteString(b.Header.Line.RawSource)
	for _, line := range b.Body {
		out.WriteString(line.RawSource)
	}
	out.WriteString(b.Tail.RawSource)
	out.WriteString(b.Terminator.RawSource)
	return out.String()
}
func (b BlockSource) BodySource() string {
	var out strings.Builder
	for _, line := range b.Body {
		out.WriteString(line.RawSource)
	}
	return out.String()
}
func (b BlockSource) Span() Span {
	end := b.Header.Line.Span.End
	if len(b.Body) > 0 {
		end = b.Body[len(b.Body)-1].Span.End
	}
	if b.Tail.RawSource != "" {
		end = b.Tail.Span.End
	}
	if b.Closed {
		end = b.Terminator.Span.End
	}
	return Span{Start: b.Header.Line.Span.Start, End: end}
}
func (b BlockSource) clone() BlockSource {
	b.Rule = cloneBlockRule(b.Rule)
	b.Body = append([]BlockLine(nil), b.Body...)
	return b
}
func cloneBlockRule(rule BlockRule) BlockRule {
	rule.Ends = append([]string(nil), rule.Ends...)
	rule.Nested = append([]BlockPair(nil), rule.Nested...)
	for i := range rule.Nested {
		rule.Nested[i].Ends = append([]string(nil), rule.Nested[i].Ends...)
	}
	return rule
}

// BlockHandler separates bounded syntax capture from interpretation. Begin
// returns a declarative rule before any body is consumed. ParseBlock is called
// once, only for a complete valid capture, with caller-owned slice metadata.
// No reader, include resolver or external execution is supplied. Handlers are
// trusted Go callbacks: they must bound their own computation and protect state.
type BlockHandler interface {
	CanHandle(cmd, pkg string) bool
	Begin(BlockHeader) (BlockRule, error)
	ParseBlock(BlockSource) (Token, error)
}

var ErrInvalidBlock = errors.New("invalid block syntax")

// BlockSyntaxError identifies an invalid header, delimiter or incomplete block.
type BlockSyntaxError struct {
	Position Position
	Message  string
	Err      error
}

func (e *BlockSyntaxError) Error() string { return e.Message }
func (e *BlockSyntaxError) Unwrap() []error {
	if e.Err == nil {
		return []error{ErrInvalidBlock}
	}
	return []error{ErrInvalidBlock, e.Err}
}
func blockSyntax(position Position, err error) error {
	return &BlockSyntaxError{Position: position, Message: err.Error(), Err: err}
}
func validBlockCommand(cmd string) bool {
	return len(cmd) > 1 && cmd[0] == '.' && utf8.ValidString(cmd) && !strings.ContainsAny(cmd, " \t\r\n\x00")
}
func validateBlockRule(rule BlockRule) error {
	if len(rule.Ends) == 0 || len(rule.Ends) > 128 || len(rule.Nested) > 128 {
		return fmt.Errorf("%w: invalid block rule size", ErrInvalidHandler)
	}
	validateEnds := func(ends []string) error {
		if len(ends) == 0 || len(ends) > 128 {
			return fmt.Errorf("%w: invalid closing commands", ErrInvalidHandler)
		}
		seen := map[string]bool{}
		for _, end := range ends {
			if !validBlockCommand(end) || seen[end] {
				return fmt.Errorf("%w: invalid/duplicate closing command %q", ErrInvalidHandler, end)
			}
			seen[end] = true
		}
		return nil
	}
	if err := validateEnds(rule.Ends); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, pair := range rule.Nested {
		if !validBlockCommand(pair.Start) || seen[pair.Start] {
			return fmt.Errorf("%w: invalid/duplicate nested opener %q", ErrInvalidHandler, pair.Start)
		}
		if err := validateEnds(pair.Ends); err != nil {
			return err
		}
		seen[pair.Start] = true
	}
	return nil
}
func normalizedBlockCommand(line string) string {
	cmd, _ := splitCommandAndArgs(line)
	if len(cmd) > 0 && (cmd[0] == '.' || cmd[0] == '\'') {
		return "." + cmd[1:]
	}
	return ""
}
func containsBlockEnd(ends []string, cmd string) bool {
	for _, end := range ends {
		if end == cmd {
			return true
		}
	}
	return false
}

type blockCapture struct {
	source  BlockSource
	stack   [][]string
	limit   int
	failure error
}

func newBlockCapture(header BlockHeader, rule BlockRule, limit int) *blockCapture {
	rule = cloneBlockRule(rule)
	return &blockCapture{source: BlockSource{Header: header, Rule: rule}, stack: [][]string{rule.Ends}, limit: limit}
}
func (c *blockCapture) consume(line BlockLine) error {
	cmd := normalizedBlockCommand(line.Text())
	match := -1
	for i := len(c.stack) - 1; i >= 0; i-- {
		if containsBlockEnd(c.stack[i], cmd) {
			match = i
			break
		}
	}
	if match >= 0 {
		if match != len(c.stack)-1 {
			c.mismatch(line, cmd)
		}
		c.stack = c.stack[:match]
		if match == 0 {
			c.source.Terminator = line
			c.source.Closed = true
			return nil
		}
	} else {
		for _, pair := range c.source.Rule.Nested {
			if containsBlockEnd(pair.Ends, cmd) {
				c.mismatch(line, cmd)
				break
			}
		}
		for _, pair := range c.source.Rule.Nested {
			if cmd == pair.Start {
				if len(c.stack) >= c.limit {
					c.source.Body = append(c.source.Body, line)
					return &DocumentInputError{Position: line.Span.Start, Err: &DocumentLimitError{Kind: "block depth", Limit: c.limit}}
				}
				c.stack = append(c.stack, pair.Ends)
				break
			}
		}
	}
	c.source.Body = append(c.source.Body, line)
	return nil
}
func (c *blockCapture) mismatch(line BlockLine, cmd string) {
	if c.failure == nil {
		c.failure = blockSyntax(line.Span.Start, fmt.Errorf("mismatched block delimiter %s (expected %s)", cmd, strings.Join(c.stack[len(c.stack)-1], " or ")))
	}
}
func (c *blockCapture) finishError() error {
	if c.source.Closed {
		return c.failure
	}
	cmd := normalizedBlockCommand(c.source.Header.Line.Text())
	label := "block " + cmd + " (expected " + strings.Join(c.stack[len(c.stack)-1], " or ") + ")"
	switch cmd {
	case ".de", ".am":
		label = "macro definition " + cmd
	case ".ig":
		label = "ignored block (.ig)"
	}
	return errors.Join(c.failure, blockSyntax(c.source.Header.Line.Span.Start, fmt.Errorf("unterminated %s", label)))
}

// TokenBlockHeader preserves a structurally captured header whose arguments do
// not fit an ordinary generated line token, including custom block headers.
type TokenBlockHeader struct{ Command, Args, Package string }

func (t *TokenBlockHeader) String() string {
	if t.Args == "" {
		return t.Command
	}
	return t.Command + " " + t.Args
}
func (t *TokenBlockHeader) TokenName() string    { return normalizedBlockCommand(t.Command) }
func (t *TokenBlockHeader) TokenKind() TokenKind { return KindBlock }
func (t *TokenBlockHeader) PackageName() string  { return t.Package }

var _ Node = (*TokenBlockHeader)(nil)

func physicalBlockLines(source string, start Position) []BlockLine {
	var lines []BlockLine
	for len(source) > 0 {
		end := strings.IndexByte(source, '\n')
		if end < 0 {
			end = len(source)
		} else {
			end++
		}
		raw := source[:end]
		next := advanceSourcePosition(start, raw)
		lines = append(lines, BlockLine{RawSource: raw, Span: Span{Start: start, End: next}})
		source = source[end:]
		start = next
	}
	return lines
}
