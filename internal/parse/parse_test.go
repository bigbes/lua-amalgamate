package parse

import (
	"testing"
)

func TestParseStaticRequire(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected []RequireInfo
	}{
		{
			name:   "simple require",
			source: `require("foo")`,
			expected: []RequireInfo{
				{Name: "foo", Line: 1, Static: true},
			},
		},
		{
			name:   "require with dots",
			source: `require("foo.bar")`,
			expected: []RequireInfo{
				{Name: "foo.bar", Line: 1, Static: true},
			},
		},
		{
			name: "multiple requires",
			source: `local a = require("a")
local b = require("b")`,
			expected: []RequireInfo{
				{Name: "a", Line: 1, Static: true},
				{Name: "b", Line: 2, Static: true},
			},
		},
		{
			name:   "pcall require",
			source: `pcall(require, "module")`,
			expected: []RequireInfo{
				{Name: "module", Line: 1, Static: true},
			},
		},
		{
			name:   "dynamic require",
			source: `require(var)`,
			expected: []RequireInfo{
				{Name: "", Line: 1, Static: false},
			},
		},
		{
			name: "require in function",
			source: `function f()
  require("inner")
end`,
			expected: []RequireInfo{
				{Name: "inner", Line: 2, Static: true},
			},
		},
		{
			name:     "no requires",
			source:   `print("hello")`,
			expected: []RequireInfo{},
		},
	}

	parser := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.Parse([]byte(tt.source), "test.lua")
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if len(got) != len(tt.expected) {
				t.Fatalf("Parse() got %d requires, want %d", len(got), len(tt.expected))
			}
			for i := range got {
				if got[i].Name != tt.expected[i].Name {
					t.Errorf("require %d: got Name = %q, want %q", i, got[i].Name, tt.expected[i].Name)
				}
				if got[i].Line != tt.expected[i].Line {
					t.Errorf("require %d: got Line = %d, want %d", i, got[i].Line, tt.expected[i].Line)
				}
				if got[i].Static != tt.expected[i].Static {
					t.Errorf("require %d: got Static = %v, want %v", i, got[i].Static, tt.expected[i].Static)
				}
			}
		})
	}
}
