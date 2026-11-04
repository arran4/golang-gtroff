package groff

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

// IncludeResolver resolves explicitly enabled .so and .mso directives.
// Each successful call transfers its reader to ParseDocument, which closes it
// after parsing if it implements io.Closer. Direct callers must likewise close
// readers they receive. Return a non-closing wrapper for a borrowed reader.
type IncludeResolver interface {
	Resolve(filename string) (io.Reader, error)
}

// SafeDirectoryIncludeResolver restricts includes to BaseDir, including symlink
// targets. MaxDepth bounds simultaneously open include readers (default 10).
// Close each returned reader to release its cycle/depth reservation. Configure
// BaseDir and MaxDepth before use; Resolve and reader Close may run concurrently.
type SafeDirectoryIncludeResolver struct {
	BaseDir  string
	MaxDepth int
	mu       sync.Mutex
	active   map[*directoryIncludeReader]os.FileInfo
}

func NewSafeDirectoryIncludeResolver(baseDir string, maxDepth int) *SafeDirectoryIncludeResolver {
	if maxDepth <= 0 {
		maxDepth = 10
	}
	return &SafeDirectoryIncludeResolver{
		BaseDir:  baseDir,
		MaxDepth: maxDepth,
	}
}

func (r *SafeDirectoryIncludeResolver) Resolve(filename string) (io.Reader, error) {
	// Reject non-local names before the OS boundary. Root.Open also enforces
	// containment while resolving symlinks; a lexical prefix check cannot do so.
	cleaned := filepath.Clean(filename)
	if !filepath.IsLocal(cleaned) {
		return nil, fmt.Errorf("access denied: path traversal not allowed: %s", filename)
	}
	root, err := os.OpenRoot(r.BaseDir)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	f, err := root.Open(cleaned)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if !info.Mode().IsRegular() {
		f.Close()
		return nil, fmt.Errorf("include is not a regular file: %s", filename)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, active := range r.active {
		if os.SameFile(active, info) {
			f.Close()
			return nil, fmt.Errorf("include cycle detected: %s", filename)
		}
	}
	limit := r.MaxDepth
	if limit <= 0 {
		limit = 10
	}
	if len(r.active) >= limit {
		f.Close()
		return nil, fmt.Errorf("maximum include depth exceeded (%d)", limit)
	}
	if r.active == nil {
		r.active = make(map[*directoryIncludeReader]os.FileInfo)
	}
	reader := &directoryIncludeReader{File: f, resolver: r}
	r.active[reader] = info
	return reader, nil
}

type directoryIncludeReader struct {
	*os.File
	resolver *SafeDirectoryIncludeResolver
	once     sync.Once
	closeErr error
}

func (r *directoryIncludeReader) Close() error {
	r.once.Do(func() {
		r.resolver.mu.Lock()
		defer r.resolver.mu.Unlock()
		r.closeErr = r.File.Close()
		delete(r.resolver.active, r)
	})
	return r.closeErr
}
