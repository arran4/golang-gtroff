package groff

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// TokenMacroDefinition represents a .de request and its multiline body.
type TokenMacroDefinition struct {
	Name string
	End  string
	Body []string
}

func (t *TokenMacroDefinition) String() string {
	var sb strings.Builder
	end := ".."
	if t.End != "" {
		end = "." + t.End
	}
	if t.End != "" {
		sb.WriteString(".de " + t.Name + " " + t.End + "\n")
	} else {
		sb.WriteString(".de " + t.Name + "\n")
	}
	for _, l := range t.Body {
		sb.WriteString(l + "\n")
	}
	sb.WriteString(end)
	return sb.String()
}

func (t *TokenMacroDefinition) TokenName() string {
	return ".de"
}

func (t *TokenMacroDefinition) TokenKind() TokenKind {
	return KindMacro
}

func (t *TokenMacroDefinition) PackageName() string {
	return "roff"
}

var _ Token = (*TokenMacroDefinition)(nil)
var _ Node = (*TokenMacroDefinition)(nil)

// TokenMacroAppend represents an .am request and its multiline body.
type TokenMacroAppend struct {
	Name string
	End  string
	Body []string
}

func (t *TokenMacroAppend) String() string {
	var sb strings.Builder
	end := ".."
	if t.End != "" {
		end = "." + t.End
	}
	if t.End != "" {
		sb.WriteString(".am " + t.Name + " " + t.End + "\n")
	} else {
		sb.WriteString(".am " + t.Name + "\n")
	}
	for _, l := range t.Body {
		sb.WriteString(l + "\n")
	}
	sb.WriteString(end)
	return sb.String()
}

func (t *TokenMacroAppend) TokenName() string {
	return ".am"
}

func (t *TokenMacroAppend) TokenKind() TokenKind {
	return KindMacro
}

func (t *TokenMacroAppend) PackageName() string {
	return "roff"
}

var _ Token = (*TokenMacroAppend)(nil)
var _ Node = (*TokenMacroAppend)(nil)

// TokenIgnoredBlock represents an .ig request and its multiline ignored body.
type TokenIgnoredBlock struct {
	End  string
	Body []string
}

func (t *TokenIgnoredBlock) String() string {
	var sb strings.Builder
	end := ".."
	if t.End != "" {
		end = "." + t.End
	}
	if t.End != "" {
		sb.WriteString(".ig " + t.End + "\n")
	} else {
		sb.WriteString(".ig\n")
	}
	for _, l := range t.Body {
		sb.WriteString(l + "\n")
	}
	sb.WriteString(end)
	return sb.String()
}

func (t *TokenIgnoredBlock) TokenName() string {
	return ".ig"
}

func (t *TokenIgnoredBlock) TokenKind() TokenKind {
	return KindBlock
}

func (t *TokenIgnoredBlock) PackageName() string {
	return "roff"
}

var _ Token = (*TokenIgnoredBlock)(nil)
var _ Node = (*TokenIgnoredBlock)(nil)

// IncludeResolver resolves file inclusion for .so and .mso directives.
type IncludeResolver interface {
	Resolve(filename string) (io.Reader, error)
}

// SafeDirectoryIncludeResolver restricts includes to a base directory and prevents directory traversal.
type SafeDirectoryIncludeResolver struct {
	BaseDir      string
	MaxDepth     int
	visitedFiles map[string]bool
}

func NewSafeDirectoryIncludeResolver(baseDir string, maxDepth int) *SafeDirectoryIncludeResolver {
	if maxDepth <= 0 {
		maxDepth = 10
	}
	return &SafeDirectoryIncludeResolver{
		BaseDir:      baseDir,
		MaxDepth:     maxDepth,
		visitedFiles: make(map[string]bool),
	}
}

func (r *SafeDirectoryIncludeResolver) Resolve(filename string) (io.Reader, error) {
	// Clean and check path traversal
	cleaned := filepath.Clean(filename)
	if strings.HasPrefix(cleaned, "../") || cleaned == ".." {
		return nil, fmt.Errorf("access denied: path traversal not allowed: %s", filename)
	}

	fullPath := filepath.Join(r.BaseDir, cleaned)
	rel, err := filepath.Rel(r.BaseDir, fullPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return nil, fmt.Errorf("access denied: path escapes base directory: %s", filename)
	}

	if r.visitedFiles[fullPath] {
		return nil, fmt.Errorf("include cycle detected: %s", filename)
	}
	if len(r.visitedFiles) >= r.MaxDepth {
		return nil, fmt.Errorf("maximum include depth exceeded (%d)", r.MaxDepth)
	}

	f, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}
	r.visitedFiles[fullPath] = true
	return f, nil
}
