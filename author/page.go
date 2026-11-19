// Package author provides small typed helpers for generating new canonical man
// documents. Literal data is escaped by the existing man writer; original source
// editing and byte-preserving serialization remain separate capabilities.
package author

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/arran4/golang-gtroff/man"
)

var (
	ErrInvalid = errors.New("invalid authoring value")
	ErrLimit   = errors.New("authoring resource limit exceeded")
)

// Limits bounds construction data and canonical output. Nonpositive selects defaults:
// 10000 blocks/sections and 8 MiB. Ceilings are 20000 blocks and 32 MiB. Bounded
// syntax/semantic validation additionally uses the existing 100000-node limits.
type Limits struct{ MaxBlocks, MaxBytes int }

// Option is a trusted constructor option. Custom options can use the public
// setters/Add methods; no reflection or dynamically typed option arguments occur.
type Option func(*Page) error

// Page is a mutable builder created by NewManPage. It is not safe for concurrent
// mutation; AST snapshots returned after generation are independent and immutable.
type Page struct {
	metadata      man.Metadata
	sections      []*Section
	limits        Limits
	blocks, bytes int
}

// Section is owned by a Page. Append/Add methods validate before mutating it;
// headings and content cannot be changed through the returned snapshots.
type Section struct {
	page    *Page
	heading string
	blocks  []Block
}

// NewManPage creates a literal man-page builder with explicit metadata. Title
// and manual section are nonempty single-line values; no clock/default date is read.
func NewManPage(title, section string, options ...Option) (*Page, error) {
	p := &Page{limits: Limits{10000, 8 << 20}}
	if err := p.SetTitle(title); err != nil {
		return nil, err
	}
	if err := p.SetSection(section); err != nil {
		return nil, err
	}
	for _, option := range options {
		if option == nil {
			return nil, fmt.Errorf("%w: nil option", ErrInvalid)
		}
		if err := option(p); err != nil {
			return nil, err
		}
	}
	return p, nil
}
func WithDate(value string) Option   { return func(p *Page) error { return p.SetDate(value) } }
func WithSource(value string) Option { return func(p *Page) error { return p.SetSource(value) } }
func WithManual(value string) Option { return func(p *Page) error { return p.SetManual(value) } }
func WithLimits(limits Limits) Option {
	return func(p *Page) error {
		if p == nil {
			return ErrInvalid
		}
		if limits.MaxBlocks <= 0 {
			limits.MaxBlocks = 10000
		}
		if limits.MaxBytes <= 0 {
			limits.MaxBytes = 8 << 20
		}
		if limits.MaxBlocks > 20000 || limits.MaxBytes > 32<<20 {
			return fmt.Errorf("%w: limits exceed supported ceilings", ErrLimit)
		}
		if p.blocks > limits.MaxBlocks || p.bytes > limits.MaxBytes {
			return ErrLimit
		}
		p.limits = limits
		return nil
	}
}
func (p *Page) SetTitle(value string) error {
	if p == nil {
		return ErrInvalid
	}
	if err := lineValue(value, true); err != nil {
		return err
	}
	p.metadata.Title = value
	return nil
}
func (p *Page) SetSection(value string) error {
	if p == nil {
		return ErrInvalid
	}
	if err := lineValue(value, true); err != nil {
		return err
	}
	p.metadata.Section = value
	return nil
}
func (p *Page) SetDate(value string) error {
	if p == nil {
		return ErrInvalid
	}
	if err := lineValue(value, false); err != nil {
		return err
	}
	p.metadata.Date = value
	return nil
}
func (p *Page) SetSource(value string) error {
	if p == nil {
		return ErrInvalid
	}
	if err := lineValue(value, false); err != nil {
		return err
	}
	p.metadata.Source = value
	return nil
}
func (p *Page) SetManual(value string) error {
	if p == nil {
		return ErrInvalid
	}
	if err := lineValue(value, false); err != nil {
		return err
	}
	p.metadata.Manual = value
	return nil
}
func (p *Page) Metadata() man.Metadata {
	if p == nil {
		return man.Metadata{}
	}
	return p.metadata
}

// AddSection appends one section and validates all supplied blocks before mutation.
func (p *Page) AddSection(heading string, blocks ...Block) (*Section, error) {
	if p == nil || p.limits.MaxBlocks == 0 {
		return nil, ErrInvalid
	}
	if err := lineValue(heading, true); err != nil {
		return nil, err
	}
	count, bytes, err := validateBlocks(blocks, false)
	if err != nil {
		return nil, err
	}
	count++
	bytes += len(heading)
	if count > p.limits.MaxBlocks-p.blocks || bytes > p.limits.MaxBytes-p.bytes {
		return nil, ErrLimit
	}
	section := &Section{page: p, heading: heading, blocks: append([]Block(nil), blocks...)}
	p.sections = append(p.sections, section)
	p.blocks += count
	p.bytes += bytes
	return section, nil
}

// Heading returns the supplied heading; canonical output can normalize flowing whitespace.
func (s *Section) Heading() string {
	if s == nil {
		return ""
	}
	return s.heading
}

// Append adds literal blocks in source flow order, with shared page limits.
func (s *Section) Append(blocks ...Block) error {
	if s == nil || s.page == nil {
		return ErrInvalid
	}
	count, bytes, err := validateBlocks(blocks, false)
	if err != nil {
		return err
	}
	p := s.page
	if count > p.limits.MaxBlocks-p.blocks || bytes > p.limits.MaxBytes-p.bytes {
		return ErrLimit
	}
	s.blocks = append(s.blocks, blocks...)
	p.blocks += count
	p.bytes += bytes
	return nil
}
func (s *Section) AddParagraph(text string) error { return s.Append(Paragraph(text)) }
func (s *Section) AddDefinition(term, description string) error {
	return s.Append(Definition(term, description))
}
func lineValue(value string, required bool) error {
	if len(value) > 4096 {
		return fmt.Errorf("%w: line value exceeds 4096 bytes", ErrLimit)
	}
	if required && strings.TrimSpace(value) == "" {
		return fmt.Errorf("%w: nonempty single-line value required", ErrInvalid)
	}
	return textValue(value, false)
}
func textValue(value string, multiline bool) error {
	if !utf8.ValidString(value) {
		return fmt.Errorf("%w: invalid UTF-8", ErrInvalid)
	}
	for _, r := range value {
		if unicode.IsControl(r) && !(r == '\t' || multiline && r == '\n') {
			return fmt.Errorf("%w: control character in literal text", ErrInvalid)
		}
	}
	return nil
}
