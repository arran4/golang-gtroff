package author

import "strings"

type blockKind uint8

const (
	paragraph blockKind = iota + 1
	definition
	preformatted
	subsection
)

// Block is an immutable literal-content value made by the constructors below.
// External helpers can return Block or []Block assembled from application data;
// no new macro switch or interface implementation is needed. A zero Block is invalid.
type Block struct {
	kind       blockKind
	text, term string
	children   []Block
}

// Paragraph contains literal flowing text. LF/CRLF are soft line boundaries;
// canonical man writing normalizes flowing whitespace, not the source spelling.
func Paragraph(text string) Block { return Block{kind: paragraph, text: normalizeLines(text)} }

// Definition contains a literal single-line term and flowing description. The
// initial convention renders the term bold. Empty descriptions are valid; lint
// policy is separate from construction validity.
func Definition(term, description string) Block {
	return Block{kind: definition, term: term, text: normalizeLines(description)}
}

// Preformatted preserves literal line layout (canonical LF). Controls/backslashes
// remain data, escaped by the existing canonical writer.
func Preformatted(text string) Block { return Block{kind: preformatted, text: normalizeLines(text)} }

// LiteralRoff displays original roff as literal preformatted data. Requests are
// escaped, never inserted as active source. This is not a raw-source bypass.
func LiteralRoff(source string) Block { return Preformatted(source) }

// Subsection groups blocks under a single subsection heading within a section.
// Following section flow remains under the latest subsection until another
// heading, as in man source. Nested subsections are rejected when appended; deeper document/layout trees
// are deliberately outside this small authoring subset.
func Subsection(heading string, blocks ...Block) Block {
	return Block{kind: subsection, term: heading, children: append([]Block(nil), blocks...)}
}
func normalizeLines(text string) string { return strings.ReplaceAll(text, "\r\n", "\n") }
func validateBlocks(blocks []Block, inSubsection bool) (count, bytes int, err error) {
	for _, block := range blocks {
		count++
		bytes += len(block.text) + len(block.term)
		switch block.kind {
		case paragraph, preformatted:
			err = textValue(block.text, true)
		case definition:
			if err = lineValue(block.term, true); err == nil {
				err = textValue(block.text, true)
			}
		case subsection:
			if inSubsection {
				return 0, 0, ErrInvalid
			}
			if err = lineValue(block.term, true); err == nil {
				var n, size int
				n, size, err = validateBlocks(block.children, true)
				count += n
				bytes += size
			}
		default:
			err = ErrInvalid
		}
		if err != nil {
			return 0, 0, err
		}
	}
	return count, bytes, nil
}
