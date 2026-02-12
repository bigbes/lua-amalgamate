package transform

import (
	"testing"
)

func TestCommentTransformer(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output string
	}{
		{
			name:   "line comment",
			input:  "print('hello') -- comment\nprint('world')",
			output: "print('hello') \nprint('world')",
		},
		{
			name:   "block comment",
			input:  "print('hello') --[[ block comment ]] print('world')",
			output: "print('hello')  print('world')",
		},
		{
			name:   "nested block comment with equals",
			input:  "print('hello') --[==[ nested ]==] print('world')",
			output: "print('hello')  print('world')",
		},
		{
			name:   "comment in string",
			input:  `str = "-- not a comment" -- actual comment`,
			output: `str = "-- not a comment" `,
		},
		{
			name:   "single quote string",
			input:  `str = '-- not a comment' -- actual comment`,
			output: `str = '-- not a comment' `,
		},
		{
			name:   "escape in string",
			input:  `str = "quote \" -- not comment" -- comment`,
			output: `str = "quote \" -- not comment" `,
		},
		{
			name:   "long string",
			input:  `str = [[-- not a comment]] -- comment`,
			output: `str = [[-- not a comment]] `,
		},
		{
			name:   "multiple comments",
			input:  "a -- c1\nb -- c2\nc",
			output: "a \nb \nc",
		},
		{
			name:   "empty input",
			input:  "",
			output: "",
		},
		{
			name:   "only comment",
			input:  "-- comment only",
			output: "",
		},
	}

	transformer := &commentTransformer{}
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
