package parse

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			require.NoError(t, err, "Parse() error")
			require.Len(t, got, len(tt.expected), "Parse() requires count")
			for i := range got {
				assert.Equal(t, tt.expected[i].Name, got[i].Name, "require %d: Name", i)
				assert.Equal(t, tt.expected[i].Line, got[i].Line, "require %d: Line", i)
				assert.Equal(t, tt.expected[i].Static, got[i].Static, "require %d: Static", i)
			}
		})
	}
}
