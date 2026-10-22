package groff

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// ArgumentLexeme holds a decoded argument and its original lexeme. Start and End
// retain the legacy half-open rune offsets; Raw preserves original bytes.
type ArgumentLexeme struct {
	Raw    string
	Value  string
	Start  int
	End    int
	Quoted bool
}

// ArgumentSpan is a half-open byte range in an ArgumentList's Source.
type ArgumentSpan struct{ Start, End int }

type argumentCapture uint8

const (
	captureArgumentValues argumentCapture = iota
	captureArgumentLexemes
	captureArgumentSource
)

// ArgumentList preserves argument source independently of decoded/canonical data.
// Its accessors return copies. A malformed quote leaves successfully scanned
// arguments available and marks the incomplete argument as Unparsed.
type ArgumentList struct {
	values                  []string
	source                  string
	lexemes                 []ArgumentLexeme
	spans                   []ArgumentSpan
	byteStop, runeStop      int
	comment, unparsed       ArgumentSpan
	hasComment, hasUnparsed bool
	err                     error
}

func (a *ArgumentList) Source() string            { return a.source }
func (a *ArgumentList) Lexemes() []ArgumentLexeme { return append([]ArgumentLexeme(nil), a.lexemes...) }
func (a *ArgumentList) ByteSpans() []ArgumentSpan { return append([]ArgumentSpan(nil), a.spans...) }

// Stop returns byte and rune offsets at the comment marker or end of source.
func (a *ArgumentList) Stop() (byteOffset, runeOffset int) { return a.byteStop, a.runeStop }
func (a *ArgumentList) Comment() (ArgumentSpan, bool)      { return a.comment, a.hasComment }
func (a *ArgumentList) Unparsed() (ArgumentSpan, bool)     { return a.unparsed, a.hasUnparsed }

// ArgumentSyntaxError locates an incomplete quoted argument, preserving
// errors.Is(err, ErrUnclosedQuote). Offsets are relative to the argument source.
type ArgumentSyntaxError struct {
	Span               ArgumentSpan
	RuneStart, RuneEnd int
}

func (e *ArgumentSyntaxError) Error() string {
	return fmt.Sprintf("%v: unterminated quote starting at index %d", ErrUnclosedQuote, e.RuneStart)
}
func (e *ArgumentSyntaxError) Unwrap() error { return ErrUnclosedQuote }

// ParseArguments scans the existing argument profile once and returns a non-nil
// source snapshot even on a syntax error. Source and Raw retain invalid UTF-8
// bytes; decoded Value changes only lexical quoting/escaped-space syntax.
func ParseArguments(source string) (*ArgumentList, error) {
	list, err := scanArguments(source, captureArgumentSource)
	return &list, err
}

// LexArguments is the compatibility lexer. Start/End and the returned stop index
// are rune offsets. It returns no lexemes on an error; ParseArguments exposes
// preserved partial arguments, byte spans, comments and malformed tails.
func LexArguments(source string) ([]ArgumentLexeme, int, error) {
	list, err := scanArguments(source, captureArgumentLexemes)
	if err != nil {
		return nil, list.runeStop, err
	}
	return list.lexemes, list.runeStop, nil
}

func scanArguments(source string, capture argumentCapture) (ArgumentList, error) {
	list := ArgumentList{source: source}
	pos, runePos := 0, 0
	advance := func() {
		_, width := utf8.DecodeRuneInString(source[pos:])
		pos += width
		runePos++
	}
	for pos < len(source) {
		for pos < len(source) && (source[pos] == ' ' || source[pos] == '\t') {
			pos++
			runePos++
		}
		if pos == len(source) {
			break
		}
		if source[pos] == '\\' && pos+1 < len(source) && source[pos+1] == '"' {
			list.comment = ArgumentSpan{pos, len(source)}
			list.hasComment = true
			break
		}
		byteStart, runeStart := pos, runePos
		quoted := source[pos] == '"'
		if quoted {
			pos++
			runePos++
		}
		closed := !quoted
		var value strings.Builder
		for pos < len(source) {
			if !quoted && (source[pos] == ' ' || source[pos] == '\t') {
				break
			}
			if !quoted && source[pos] == '\\' && pos+1 < len(source) && source[pos+1] == '"' {
				break
			}
			if source[pos] == '\\' {
				start := pos
				pos++
				runePos++
				if pos == len(source) {
					value.WriteByte('\\')
					break
				}
				next := source[pos]
				advance()
				if next == ' ' || quoted && next == '"' {
					value.WriteByte(next)
				} else {
					value.WriteString(source[start:pos])
				}
				continue
			}
			if quoted && source[pos] == '"' {
				pos++
				runePos++
				if pos < len(source) && source[pos] == '"' {
					value.WriteByte('"')
					pos++
					runePos++
					continue
				}
				closed = true
				break
			}
			start := pos
			advance()
			value.WriteString(source[start:pos])
		}
		if !closed {
			list.byteStop, list.runeStop = pos, runePos
			list.unparsed = ArgumentSpan{byteStart, len(source)}
			list.hasUnparsed = true
			list.err = &ArgumentSyntaxError{Span: list.unparsed, RuneStart: runeStart, RuneEnd: runePos}
			return list, list.err
		}
		if capture == captureArgumentValues {
			list.values = append(list.values, value.String())
		} else {
			raw := source[byteStart:pos]
			if capture == captureArgumentLexemes {
				raw = strings.Clone(raw)
			}
			list.lexemes = append(list.lexemes, ArgumentLexeme{Raw: raw, Value: value.String(), Start: runeStart, End: runePos, Quoted: quoted})
			if capture == captureArgumentSource {
				list.spans = append(list.spans, ArgumentSpan{byteStart, pos})
			}
		}
	}
	list.byteStop, list.runeStop = pos, runePos
	return list, nil
}

// ArgumentParser retains decoded values and rune-index/error-text compatibility
// while using the same scanner as LexArguments and ParseArguments.
func ArgumentParser(source string) ([]string, int, error) {
	list, err := scanArguments(source, captureArgumentValues)
	if err != nil {
		return nil, list.runeStop, argumentParserError{err}
	}
	return list.values, list.runeStop, nil
}

type argumentParserError struct{ cause error }

func (e argumentParserError) Error() string { return ErrUnclosedQuote.Error() + `: stack "` }
func (e argumentParserError) Unwrap() error { return e.cause }
