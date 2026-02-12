package transform

import (
	"testing"
)

func TestEmptyLineTransformer(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output string
	}{
		{
			name:   "empty lines",
			input:  "line1\n\nline2\n  \nline3",
			output: "line1\nline2\nline3",
		},
		{
			name:   "trailing newline",
			input:  "line1\nline2\n",
			output: "line1\nline2\n",
		},
		{
			name:   "no trailing newline",
			input:  "line1\nline2",
			output: "line1\nline2",
		},
		{
			name:   "only whitespace lines",
			input:  "  \n\t\n  \n",
			output: "",
		},
		{
			name:   "mixed",
			input:  "a\n\nb\n  \nc\n\nd",
			output: "a\nb\nc\nd",
		},
		{
			name:   "empty input",
			input:  "",
			output: "",
		},
	}

	transformer := &emptyLineTransformer{}
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
