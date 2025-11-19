package groff

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseLine_Macros(t *testing.T) {
	tests := []struct {
		name string
		line string
		want Token
	}{
		{
			name: "TH",
			line: `.TH foo 1 2023-10-26 "GNU" "GNU Utilities"`,
			want: &TokenMacroTH{
				Title:   "foo",
				Section: "1",
				Extra1:  "2023-10-26",
				Extra2:  "GNU",
				Extra3:  "GNU Utilities",
			},
		},
		{
			name: "SH",
			line: `.SH Name`,
			want: &TokenMacroSH{
				Heading: "Name",
			},
		},
		{
			name: "SS",
			line: `.SS "Subsection Title"`,
			want: &TokenMacroSS{
				Heading: "Subsection Title",
			},
		},
		{
			name: "TP",
			line: `.TP`,
			want: &TokenMacroTP{
				Nnn: "",
			},
		},
		{
			name: "IP",
			line: `.IP \(bu 2`,
			want: &TokenMacroIP{
				Designator: `\(bu`,
				Nnn:        "2",
			},
		},
		{
			name: "B",
			line: `.B bold`,
			want: &TokenMacroB{
				Text: "bold",
			},
		},
		{
			name: "I",
			line: `.I italic`,
			want: &TokenMacroI{
				Text: "italic",
			},
		},
		{
			name: "BI",
			line: `.BI bold italic`,
			want: &TokenMacroBI{
				Text: "bold italic",
			},
		},
		{
			name: "BR",
			line: `.BR bold roman`,
			want: &TokenMacroBR{
				Text: "bold roman",
			},
		},
		{
			name: "IB",
			line: `.IB italic bold`,
			want: &TokenMacroIB{
				Text: "italic bold",
			},
		},
		{
			name: "IR",
			line: `.IR italic roman`,
			want: &TokenMacroIR{
				Text: "italic roman",
			},
		},
		{
			name: "RB",
			line: `.RB roman bold`,
			want: &TokenMacroRB{
				Text: "roman bold",
			},
		},
		{
			name: "RI",
			line: `.RI roman italic`,
			want: &TokenMacroRI{
				Text: "roman italic",
			},
		},
		{
			name: "SM",
			line: `.SM small`,
			want: &TokenMacroSM{
				Text: "small",
			},
		},
		{
			name: "SB",
			line: `.SB smallbold`,
			want: &TokenMacroSB{
				Text: "smallbold",
			},
		},
		{
			name: "OP",
			line: `.OP -f file`,
			want: &TokenMacroOP{
				Optionname:     "-f",
				Optionargument: "file",
			},
		},
		{
			name: "SY",
			line: `.SY command`,
			want: &TokenMacroSY{
				Command: "command",
			},
		},
		{
			name: "YS",
			line: `.YS`,
			want: &TokenMacroYS{},
		},
		{
			name: "MR",
			line: `.MR man 1`,
			want: &TokenMacroMR{
				Topic:         "man",
				Manualsection: "1",
			},
		},
		{
			name: "MT",
			line: `.MT user@example.com`,
			want: &TokenMacroMT{
				Address: "user@example.com",
			},
		},
		{
			name: "ME",
			line: `.ME`,
			want: &TokenMacroME{},
		},
		{
			name: "UR",
			line: `.UR http://example.com`,
			want: &TokenMacroUR{
				Uri: "http://example.com",
			},
		},
		{
			name: "UE",
			line: `.UE`,
			want: &TokenMacroUE{},
		},
		{
			name: "EX",
			line: `.EX`,
			want: &TokenMacroEX{
				Indent: "",
			},
		},
		{
			name: "EE",
			line: `.EE`,
			want: &TokenMacroEE{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLine(tt.line)
			if err != nil {
				t.Errorf("ParseLine() error = %v", err)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("ParseLine() mismatch (-got +want):\n%s", diff)
			}
		})
	}
}
