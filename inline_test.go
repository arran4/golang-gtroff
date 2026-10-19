package groff

import (
	"strings"
	"testing"
)

func TestParseInline(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []InlineToken
	}{
		{
			name:  "plain text",
			input: "Hello world!",
			expected: []InlineToken{
				{Kind: InlineText, Raw: "Hello world!", Value: "Hello world!"},
			},
		},
		{
			name:  "string reference bracketed",
			input: "Title: \\*[HF] Heading",
			expected: []InlineToken{
				{Kind: InlineText, Raw: "Title: ", Value: "Title: "},
				{Kind: InlineStringRef, Raw: "\\*[HF]", Name: "HF"},
				{Kind: InlineText, Raw: " Heading", Value: " Heading"},
			},
		},
		{
			name:  "register reference bracketed",
			input: "Offset: \\n[PO] pt",
			expected: []InlineToken{
				{Kind: InlineText, Raw: "Offset: ", Value: "Offset: "},
				{Kind: InlineRegisterRef, Raw: "\\n[PO]", Name: "PO"},
				{Kind: InlineText, Raw: " pt", Value: " pt"},
			},
		},
		{
			name:  "bullet special character two-char",
			input: "Item \\(bu First",
			expected: []InlineToken{
				{Kind: InlineText, Raw: "Item ", Value: "Item "},
				{Kind: InlineSpecialChar, Raw: "\\(bu", Name: "bu"},
				{Kind: InlineText, Raw: " First", Value: " First"},
			},
		},
		{
			name:  "bullet special character bracketed",
			input: "Item \\[bu] First",
			expected: []InlineToken{
				{Kind: InlineText, Raw: "Item ", Value: "Item "},
				{Kind: InlineSpecialChar, Raw: "\\[bu]", Name: "bu"},
				{Kind: InlineText, Raw: " First", Value: " First"},
			},
		},
		{
			name:  "font and size change",
			input: "\\fBbold\\fR and \\s[12]big\\s[10]",
			expected: []InlineToken{
				{Kind: InlineFontChange, Raw: "\\fB", Name: "B"},
				{Kind: InlineText, Raw: "bold", Value: "bold"},
				{Kind: InlineFontChange, Raw: "\\fR", Name: "R"},
				{Kind: InlineText, Raw: " and ", Value: " and "},
				{Kind: InlineSizeChange, Raw: "\\s[12]", Name: "12"},
				{Kind: InlineText, Raw: "big", Value: "big"},
				{Kind: InlineSizeChange, Raw: "\\s[10]", Name: "10"},
			},
		},
		{
			name:  "lone backslash and escaped backslash",
			input: "path\\to\\\\file\\",
			expected: []InlineToken{
				{Kind: InlineText, Raw: "path", Value: "path"},
				{Kind: InlineEscape, Raw: "\\t", Name: "t"},
				{Kind: InlineText, Raw: "o", Value: "o"},
				{Kind: InlineEscape, Raw: "\\\\", Name: "\\"},
				{Kind: InlineText, Raw: "file\\", Value: "file\\"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := ParseInline(tt.input)
			if err != nil {
				t.Fatalf("ParseInline error: %v", err)
			}

			// Verify round-trip property
			var roundTrip strings.Builder
			for _, tok := range tokens {
				roundTrip.WriteString(tok.String())
			}
			if roundTrip.String() != tt.input {
				t.Errorf("Round-trip failed: got %q, want %q", roundTrip.String(), tt.input)
			}

			if len(tokens) != len(tt.expected) {
				t.Fatalf("token count mismatch: got %d, want %d", len(tokens), len(tt.expected))
			}

			for i := range tokens {
				if tokens[i].Kind != tt.expected[i].Kind {
					t.Errorf("[%d] Kind mismatch: got %v, want %v", i, tokens[i].Kind, tt.expected[i].Kind)
				}
				if tokens[i].Raw != tt.expected[i].Raw {
					t.Errorf("[%d] Raw mismatch: got %q, want %q", i, tokens[i].Raw, tt.expected[i].Raw)
				}
				if tt.expected[i].Name != "" && tokens[i].Name != tt.expected[i].Name {
					t.Errorf("[%d] Name mismatch: got %q, want %q", i, tokens[i].Name, tt.expected[i].Name)
				}
			}
		})
	}
}

func TestDecodeDefaultSpecialChar(t *testing.T) {
	cases := map[string]string{
		"bu": "•",
		"em": "—",
		"co": "©",
		"rg": "®",
	}
	for name, want := range cases {
		got, ok := DecodeDefaultSpecialChar(name)
		if !ok || got != want {
			t.Errorf("DecodeDefaultSpecialChar(%q) = %q, %v; want %q, true", name, got, ok, want)
		}
	}
}
