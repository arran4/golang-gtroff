package gen

import (
	"github.com/google/go-cmp/cmp"
	"testing"
)

func Test_parseArgRun(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		want     *ArgStruct
		wantErr  bool
		Forwards bool
	}{
		{
			name:    "Empty",
			in:      "",
			want:    &ArgStruct{},
			wantErr: false,
		},
		{
			name:    "1 required",
			in:      "a",
			want:    &ArgStruct{Words: []*Word{{"a", false}}},
			wantErr: false,
		},
		{
			name: "1 optional",
			in:   "[a]",
			want: &ArgStruct{
				Words: []*Word{{"a", true}},
				OptionalRanges: [][2]int{
					{0, 1},
				},
			},
			Forwards: true,
			wantErr:  false,
		},
		{
			name: "1 required 1 optional",
			in:   "a [b]",
			want: &ArgStruct{
				Words: []*Word{{"a", false}, {"b", true}},
				OptionalRanges: [][2]int{
					{1, 2},
				},
			},
			Forwards: true,
			wantErr:  false,
		},
		{
			name: "2 optional chained",
			in:   "[a [b]]",
			want: &ArgStruct{
				Words: []*Word{{"a", true}, {"b", true}},
				OptionalRanges: [][2]int{
					{0, 2},
					{1, 2},
				},
			},
			Forwards: true,
			wantErr:  false,
		},
		{
			name: "2 optional alternative",
			in:   "[a] [b]",
			want: &ArgStruct{
				Words: []*Word{{"a", true}, {"b", true}},
				OptionalRanges: [][2]int{
					{0, 1},
					{1, 2},
				},
			},
			Forwards: true,
			wantErr:  false,
		},
		{
			name: "2 same name",
			in:   "a a",
			want: &ArgStruct{
				Words: []*Word{{"a", false}, {"a", false}},
			},
			wantErr: false,
		},
		{
			name: ".Pn",
			in:   "[punct] path [punct]",
			want: &ArgStruct{
				Words: []*Word{{"punct", true}, {"path", false}, {"punct", true}},
				OptionalRanges: [][2]int{
					{0, 1},
					{2, 3},
				},
			},
			wantErr: false,
		},
		{
			name: "close mid word ignored",
			in:   "as]df",
			want: &ArgStruct{
				Words: []*Word{{"as]df", false}},
			},
			wantErr: false,
		},
		{
			name: "open mid word ignored",
			in:   "as[df",
			want: &ArgStruct{
				Words: []*Word{{"as[df", false}},
			},
			wantErr: false,
		},
		{
			name: "Multiple spaces either side ignored",
			in:   "  asdf   ",
			want: &ArgStruct{
				Words: []*Word{{"asdf", false}},
			},
			wantErr: false,
		},
		{
			name: "Multiple spaces between ignored",
			in:   "sdf    sdf",
			want: &ArgStruct{
				Words: []*Word{{"sdf", false}, {"sdf", false}},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseArgRun(tt.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseArgRun() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if d := cmp.Diff(got, tt.want); d != "" {
				t.Errorf("ParseArgs()\n%s", d)
			}
			if d := cmp.Diff(got.Forwards(), tt.Forwards); d != "" {
				t.Errorf("Forwards() = %s", d)
			}
		})
	}
}
