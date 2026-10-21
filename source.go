package groff

// advanceSourcePosition advances byte offsets and Unicode rune columns. Tabs
// occupy one column; newline starts the following line at column one.
func advanceSourcePosition(p Position, s string) Position {
	p.Offset += len(s)
	for _, r := range s {
		if r == '\n' {
			p.Line++
			p.Column = 1
		} else {
			p.Column++
		}
	}
	return p
}
