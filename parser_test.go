package groff

import (
	"github.com/google/go-cmp/cmp"
	"testing"
)

func TestParseLine(t *testing.T) {
	tests := []struct {
		name string
		line string
		want Token
	}{
		{
			name: "comment",
			line: `.\" This manual is distributed in the hope that it will be useful,`,
			want: TokenComment("This manual is distributed in the hope that it will be useful,"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLine(tt.line)
			if err != nil {
				t.Error(err)
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("ParseLine() = \n%s", diff)
			}
		})
	}
}
