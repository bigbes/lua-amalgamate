package transform

import (
	"testing"
)

func TestMinifyTransformer(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output string
	}{
		{
			name:   "comments and spaces",
			input:  "local  x  =  5  -- comment\nprint(x)",
			output: "local x = 5\nprint(x)",
		},
		{
			name:   "multiple spaces",
			input:  "a     b     c",
			output: "a b c",
		},
		{
			name:   "tabs and spaces",
			input:  "a \t  b\t\tc",
			output: "a b c",
		},
		{
			name:   "spaces in string",
			input:  `str = "a     b"`,
			output: `str = "a     b"`,
		},
		{
			name:   "trailing space before newline",
			input:  "line1   \nline2",
			output: "line1\nline2",
		},
		{
			name:   "block comment",
			input:  "a--[[comment]]b",
			output: "ab",
		},
		{
			name:   "empty lines",
			input:  "a\n\nb\n  \nc",
			output: "a\nb\nc",
		},
	}

	transformer := &minifyTransformer{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transformer.Transform([]byte(tt.input))
			if err != nil {
				t.Fatalf("Transform() error = %v", err)
			}
			if string(got) != tt.output {
				t.Errorf("Transform() = %q, want %q", string(got), tt.output)
			}
		})
	}
}
