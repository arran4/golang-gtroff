package groff

import (
	"strings"
)

// TokenTableBlock represents a tbl(1) block (.TS ... .TE).
type TokenTableBlock struct {
	Options string
	Lines   []string
}

func (t *TokenTableBlock) String() string {
	var sb strings.Builder
	if t.Options != "" {
		sb.WriteString(".TS " + t.Options + "\n")
	} else {
		sb.WriteString(".TS\n")
	}
	for _, l := range t.Lines {
		sb.WriteString(l + "\n")
	}
	sb.WriteString(".TE")
	return sb.String()
}

func (t *TokenTableBlock) TokenName() string {
	return ".TS"
}

func (t *TokenTableBlock) TokenKind() TokenKind {
	return KindBlock
}

func (t *TokenTableBlock) PackageName() string {
	return "roff"
}

var _ Token = (*TokenTableBlock)(nil)
var _ Node = (*TokenTableBlock)(nil)

// TokenEquationBlock represents an eqn(1) block (.EQ ... .EN).
type TokenEquationBlock struct {
	Args  string
	Lines []string
}

func (t *TokenEquationBlock) String() string {
	var sb strings.Builder
	if t.Args != "" {
		sb.WriteString(".EQ " + t.Args + "\n")
	} else {
		sb.WriteString(".EQ\n")
	}
	for _, l := range t.Lines {
		sb.WriteString(l + "\n")
	}
	sb.WriteString(".EN")
	return sb.String()
}

func (t *TokenEquationBlock) TokenName() string {
	return ".EQ"
}

func (t *TokenEquationBlock) TokenKind() TokenKind {
	return KindBlock
}

func (t *TokenEquationBlock) PackageName() string {
	return "roff"
}

var _ Token = (*TokenEquationBlock)(nil)
var _ Node = (*TokenEquationBlock)(nil)

// TokenPictureBlock represents a pic(1) block (.PS ... .PE).
type TokenPictureBlock struct {
	Args  string
	Lines []string
}

func (t *TokenPictureBlock) String() string {
	var sb strings.Builder
	if t.Args != "" {
		sb.WriteString(".PS " + t.Args + "\n")
	} else {
		sb.WriteString(".PS\n")
	}
	for _, l := range t.Lines {
		sb.WriteString(l + "\n")
	}
	sb.WriteString(".PE")
	return sb.String()
}

func (t *TokenPictureBlock) TokenName() string {
	return ".PS"
}

func (t *TokenPictureBlock) TokenKind() TokenKind {
	return KindBlock
}

func (t *TokenPictureBlock) PackageName() string {
	return "roff"
}

var _ Token = (*TokenPictureBlock)(nil)
var _ Node = (*TokenPictureBlock)(nil)

// TokenReferenceBlock represents a refer(1) bibliographic block (.[ ... .]).
type TokenReferenceBlock struct {
	Lines []string
}

func (t *TokenReferenceBlock) String() string {
	var sb strings.Builder
	sb.WriteString(".[\n")
	for _, l := range t.Lines {
		sb.WriteString(l + "\n")
	}
	sb.WriteString(".]")
	return sb.String()
}

func (t *TokenReferenceBlock) TokenName() string {
	return ".["
}

func (t *TokenReferenceBlock) TokenKind() TokenKind {
	return KindBlock
}

func (t *TokenReferenceBlock) PackageName() string {
	return "roff"
}

var _ Token = (*TokenReferenceBlock)(nil)
var _ Node = (*TokenReferenceBlock)(nil)
