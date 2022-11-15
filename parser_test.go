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

func TestArgumentParser(t *testing.T) {
	tests := []struct {
		name    string
		l       string
		want    []string
		wantErr bool
		i       int
	}{
		{
			name:    "this \"word and\" that",
			l:       "this \"word and\" that",
			want:    []string{"this", "word and", "that"},
			wantErr: false,
			i:       len("this \"word and\" that"),
		},
		{
			name:    "do-nothing  \\\" activate blank line macro",
			l:       "do-nothing  \\\" activate blank line macro",
			want:    []string{"do-nothing"},
			wantErr: false,
			i:       len("do-nothing  "),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, i, err := ArgumentParser(tt.l)
			if (err != nil) != tt.wantErr {
				t.Errorf("ArgumentParser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("ArgumentParser() got \n %s", diff)
			}
			if diff := cmp.Diff(i, tt.i); diff != "" {
				t.Errorf("ArgumentParser() i \n %s", diff)
			}
		})
	}
}
