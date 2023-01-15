package groff

import (
	"github.com/google/go-cmp/cmp"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name       string
		in         string
		wantResult []*ArgStruct
		wantErr    bool
	}{
		{
			name:       "Empty",
			in:         "",
			wantResult: []*ArgStruct{},
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
					Next:     []*ArgStruct{},
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
					Next:     []*ArgStruct{},
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
							Next:     []*ArgStruct{},
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
							Next:     []*ArgStruct{},
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
							Next:     []*ArgStruct{},
						},
					},
				},
				{
					Name:     "b",
					Optional: true,
					Position: 0,
					Next:     []*ArgStruct{},
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
					Optional: true,
					Position: 0,
					Next:     []*ArgStruct{},
				},
				{
					Name:     "a",
					Optional: true,
					Position: 1,
					Next:     []*ArgStruct{},
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
									Next:     []*ArgStruct{},
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
							Next:     []*ArgStruct{},
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
			if d := cmp.Diff(gotResult, tt.wantResult); d != "" {
				t.Errorf("ParseArgs()\n%s", d)
			}
			if d := cmp.Diff(gotResult.String(), tt.in); d != "" {
				t.Errorf(".String()\n%s", d)
			}
		})
	}
}
