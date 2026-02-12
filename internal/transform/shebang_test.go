package transform

import (
	"testing"
)

func TestShebangTransformer(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output string
	}{
		{
			name:   "shebang line",
			input:  "#!/usr/bin/env lua\nprint('hello')",
			output: "print('hello')",
		},
		{
			name:   "shebang with leading whitespace",
			input:  "   #!/usr/bin/lua\nprint('hello')",
			output: "print('hello')",
		},
		{
			name:   "shebang with extra spaces after",
			input:  "#! lua\nprint('hello')",
			output: "print('hello')",
		},
		{
			name:   "no shebang",
			input:  "print('hello')",
			output: "print('hello')",
		},
		{
			name:   "shebang inside string",
			input:  `str = "#!/usr/bin/lua"`,
			output: `str = "#!/usr/bin/lua"`,
		},
		{
			name:   "multiple shebang lines only first removed",
			input:  "#!/bin/lua\n#!/second\nprint('hi')",
			output: "#!/second\nprint('hi')",
		},
		{
			name:   "shebang with Windows line endings",
			input:  "#!/usr/bin/env lua\r\nprint('hello')",
			output: "print('hello')",
		},
		{
			name:   "empty file",
			input:  "",
			output: "",
		},
		{
			name:   "only shebang line",
			input:  "#!/usr/bin/env lua",
			output: "",
		},
	}

	transformer := &shebangTransformer{}
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

func TestMinifyTransformerRemovesShebang(t *testing.T) {
	transformer := &minifyTransformer{}
	input := "#!/usr/bin/env lua\nprint('hello')"
	got, err := transformer.Transform([]byte(input))
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}
	expected := "print('hello')"
	if string(got) != expected {
		t.Errorf("Transform() = %q, want %q", string(got), expected)
	}
}
