package gen

import (
	"github.com/google/go-cmp/cmp"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name       string
		in         string
		wantResult Args
		wantErr    bool
	}{
		{
			name:       "Empty",
			in:         "",
			wantResult: nil,
			wantErr:    false,
		},
		{
			name: "1 required",
			in:   "a",
			wantResult: []*ArgStruct{
				{
					Name:     "a",
					Optional: false,
					Position: 0,
				},
			},
			wantErr: false,
		},
		{
			name: "1 optional",
			in:   "[a]",
			wantResult: []*ArgStruct{
				{
					Name:     "a",
					Optional: true,
					Position: 0,
				},
			},
			wantErr: false,
		},
		{
			name: "1 required 1 optional",
			in:   "a [b]",
			wantResult: []*ArgStruct{
				{
					Name:     "a",
					Optional: false,
					Position: 0,
					Next: []*ArgStruct{
						{
							Name:     "b",
							Optional: true,
							Position: 1,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "2 optional chained",
			in:   "[a [b]]",
			wantResult: []*ArgStruct{
				{
					Name:     "a",
					Optional: true,
					Position: 0,
					Next: []*ArgStruct{
						{
							Name:     "b",
							Optional: true,
							Position: 1,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "2 optional alternative",
			in:   "[a] [b]",
			wantResult: []*ArgStruct{
				{
					Name:     "a",
					Optional: true,
					Position: 0,
					Next: []*ArgStruct{
						{
							Name:     "b",
							Optional: true,
							Position: 1,
						},
					},
				},
				{
					Name:     "b",
					Optional: true,
					Position: 0,
				},
			},
			wantErr: false,
		},
		{
			name: "2 same name",
			in:   "a a",
			wantResult: []*ArgStruct{
				{
					Name:     "a",
					Optional: false,
					Position: 0,
					Next: []*ArgStruct{
						{
							Name:     "a",
							Optional: false,
							Position: 1,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: ".Pn",
			in:   "[punct] path [punct]",
			wantResult: []*ArgStruct{
				{
					Name:     "punct",
					Optional: true,
					Position: 0,
					Next: []*ArgStruct{
						{
							Name:     "path",
							Optional: false,
							Position: 1,
							Next: []*ArgStruct{
								{
									Name:     "punct",
									Optional: true,
									Position: 2,
								},
							},
						},
					},
				},
				{
					Name:     "path",
					Optional: false,
					Position: 0,
					Next: []*ArgStruct{
						{
							Name:     "punct",
							Optional: true,
							Position: 1,
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, err := ParseArgs(tt.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if d := cmp.Diff(([]*ArgStruct)(gotResult), ([]*ArgStruct)(tt.wantResult)); d != "" {
				t.Errorf("ParseArgs()\n%s", d)
			}
			if d := cmp.Diff(gotResult.String(), tt.in); d != "" {
				t.Errorf(".String()\n%s", d)
			}
		})
	}
}

func Test_parseArgRun(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    *argRun
		wantErr bool
	}{
		{
			name:    "Empty",
			in:      "",
			want:    &argRun{},
			wantErr: false,
		},
		{
			name:    "1 required",
			in:      "a",
			want:    &argRun{Words: []string{"a"}},
			wantErr: false,
		},
		{
			name: "1 optional",
			in:   "[a]",
			want: &argRun{
				Words: []string{"a"},
				OptionalRanges: [][2]int{
					{0, 1},
				},
			},
			wantErr: false,
		},
		{
			name: "1 required 1 optional",
			in:   "a [b]",
			want: &argRun{
				Words: []string{"a", "b"},
				OptionalRanges: [][2]int{
					{1, 2},
				},
			},
			wantErr: false,
		},
		{
			name: "2 optional chained",
			in:   "[a [b]]",
			want: &argRun{
				Words: []string{"a", "b"},
				OptionalRanges: [][2]int{
					{0, 2},
					{1, 2},
				},
			},
			wantErr: false,
		},
		{
			name: "2 optional alternative",
			in:   "[a] [b]",
			want: &argRun{
				Words: []string{"a", "b"},
				OptionalRanges: [][2]int{
					{0, 1},
					{1, 2},
				},
			},
			wantErr: false,
		},
		{
			name:    "2 same name",
			in:      "a a",
			want:    &argRun{Words: []string{"a", "a"}},
			wantErr: false,
		},
		{
			name: ".Pn",
			in:   "[punct] path [punct]",
			want: &argRun{
				Words: []string{"punct", "path", "punct"},
				OptionalRanges: [][2]int{
					{0, 1},
					{2, 3},
				},
			},
			wantErr: false,
		},
		{
			name:    "close mid word ignored",
			in:      "as]df",
			want:    &argRun{Words: []string{"as]df"}},
			wantErr: false,
		},
		{
			name:    "open mid word ignored",
			in:      "as[df",
			want:    &argRun{Words: []string{"as[df"}},
			wantErr: false,
		},
		{
			name:    "Multiple spaces either side ignored",
			in:      "  asdf   ",
			want:    &argRun{Words: []string{"asdf"}},
			wantErr: false,
		},
		{
			name:    "Multiple spaces between ignored",
			in:      "sdf    sdf",
			want:    &argRun{Words: []string{"sdf", "sdf"}},
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
		})
	}
}
