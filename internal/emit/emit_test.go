package emit

import (
	"bytes"
	"strings"
	"testing"

	"github.com/bigbes/lua-amalgamate/internal/graph"
	"github.com/bigbes/lua-amalgamate/internal/parse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmitSingleModule(t *testing.T) {
	g := &graph.Graph{
		Modules: []*graph.Module{
			{
				ID:       0,
				FilePath: "/test/main.lua",
				Names:    []string{"main"},
				Source:   []byte("print('hello')"),
				Requires: []parse.RequireInfo{},
			},
		},
		ByPath: map[string]*graph.Module{
			"/test/main.lua": {
				ID:       0,
				FilePath: "/test/main.lua",
				Names:    []string{"main"},
				Source:   []byte("print('hello')"),
				Requires: []parse.RequireInfo{},
			},
		},
		Entry: &graph.Module{
			ID:       0,
			FilePath: "/test/main.lua",
			Names:    []string{"main"},
			Source:   []byte("print('hello')"),
			Requires: []parse.RequireInfo{},
		},
		Warnings: []graph.Warning{},
	}

	var buf bytes.Buffer
	require.NoError(t, Emit(&buf, g, nil, Options{}))

	output := buf.String()
	expectedLines := []string{
		"-- Amalgamated by lua-amalgamate",
		"-- Entry: main",
		"package.preload[\"main\"] = function(...)",
		"  print('hello')",
		"end",
		"require(\"main\")",
	}
	for _, line := range expectedLines {
		assert.Contains(t, output, line, "Output missing line %q", line)
	}
}

func TestEmitMultiModule(t *testing.T) {
	g := &graph.Graph{
		Modules: []*graph.Module{
			{
				ID:       0,
				FilePath: "/test/main.lua",
				Names:    []string{"main"},
				Source:   []byte("require('a')"),
				Requires: []parse.RequireInfo{{Name: "a", Line: 1, Static: true}},
			},
			{
				ID:       1,
				FilePath: "/test/a.lua",
				Names:    []string{"a"},
				Source:   []byte("print('a')"),
				Requires: []parse.RequireInfo{},
			},
		},
		ByPath: map[string]*graph.Module{
			"/test/main.lua": {
				ID:       0,
				FilePath: "/test/main.lua",
				Names:    []string{"main"},
				Source:   []byte("require('a')"),
				Requires: []parse.RequireInfo{{Name: "a", Line: 1, Static: true}},
			},
			"/test/a.lua": {
				ID:       1,
				FilePath: "/test/a.lua",
				Names:    []string{"a"},
				Source:   []byte("print('a')"),
				Requires: []parse.RequireInfo{},
			},
		},
		Entry: &graph.Module{
			ID:       0,
			FilePath: "/test/main.lua",
			Names:    []string{"main"},
			Source:   []byte("require('a')"),
			Requires: []parse.RequireInfo{{Name: "a", Line: 1, Static: true}},
		},
		Warnings: []graph.Warning{},
	}

	var buf bytes.Buffer
	require.NoError(t, Emit(&buf, g, nil, Options{}))

	output := buf.String()
	// Should contain both modules
	assert.Contains(t, output, "package.preload[\"main\"]", "Output missing main module")
	assert.Contains(t, output, "package.preload[\"a\"]", "Output missing a module")
	assert.Contains(t, output, "require(\"main\")", "Output missing entry require")
}

func TestEmitModuleWithAliases(t *testing.T) {
	g := &graph.Graph{
		Modules: []*graph.Module{
			{
				ID:       0,
				FilePath: "/test/foo.lua",
				Names:    []string{"foo", "foo.bar", "foo/bar"},
				Source:   []byte("return {}"),
				Requires: []parse.RequireInfo{},
			},
		},
		ByPath: map[string]*graph.Module{
			"/test/foo.lua": {
				ID:       0,
				FilePath: "/test/foo.lua",
				Names:    []string{"foo", "foo.bar", "foo/bar"},
				Source:   []byte("return {}"),
				Requires: []parse.RequireInfo{},
			},
		},
		Entry: &graph.Module{
			ID:       0,
			FilePath: "/test/foo.lua",
			Names:    []string{"foo"},
			Source:   []byte("return {}"),
			Requires: []parse.RequireInfo{},
		},
		Warnings: []graph.Warning{},
	}

	var buf bytes.Buffer
	require.NoError(t, Emit(&buf, g, nil, Options{}))

	output := buf.String()
	// Primary name carries the real loader; aliases delegate to it so the
	// module body runs exactly once regardless of which name is required.
	assert.Contains(t, output, "package.preload[\"foo\"] = function(...)", "Output missing primary loader for foo")
	assert.Contains(t, output, "package.preload[\"foo.bar\"] = function(...) return require(\"foo\") end", "Output missing delegating alias foo.bar")
	assert.Contains(t, output, "package.preload[\"foo/bar\"] = function(...) return require(\"foo\") end", "Output missing delegating alias foo/bar")
}

func TestEmitSuffix(t *testing.T) {
	g := &graph.Graph{
		Modules: []*graph.Module{
			{
				ID:       0,
				FilePath: "/test/main.lua",
				Names:    []string{"main"},
				Source:   []byte("print('hello')"),
				Requires: []parse.RequireInfo{},
			},
		},
		ByPath: map[string]*graph.Module{
			"/test/main.lua": {
				ID:       0,
				FilePath: "/test/main.lua",
				Names:    []string{"main"},
				Source:   []byte("print('hello')"),
				Requires: []parse.RequireInfo{},
			},
		},
		Entry: &graph.Module{
			ID:       0,
			FilePath: "/test/main.lua",
			Names:    []string{"main"},
			Source:   []byte("print('hello')"),
			Requires: []parse.RequireInfo{},
		},
		Warnings: []graph.Warning{},
	}

	var buf bytes.Buffer
	suffixCode := "print('suffix')"
	require.NoError(t, Emit(&buf, g, nil, Options{Suffix: suffixCode}))

	output := buf.String()
	assert.Contains(t, output, "print('suffix')", "Output missing suffix code")
	// Ensure suffix appears after require
	requireIdx := strings.Index(output, "require(\"main\")")
	suffixIdx := strings.Index(output, "print('suffix')")
	assert.False(t, requireIdx == -1 || suffixIdx == -1 || suffixIdx < requireIdx, "Suffix code should appear after require")
}

func TestEmitFallback(t *testing.T) {
	mod := &graph.Module{
		ID:       0,
		FilePath: "/test/main.lua",
		Names:    []string{"main"},
		Source:   []byte("print('hello')"),
		Requires: []parse.RequireInfo{},
	}
	g := &graph.Graph{
		Modules:  []*graph.Module{mod},
		ByPath:   map[string]*graph.Module{"/test/main.lua": mod},
		Entry:    mod,
		Warnings: []graph.Warning{},
	}

	var buf bytes.Buffer
	require.NoError(t, Emit(&buf, g, nil, Options{Fallback: true}))

	output := buf.String()
	// Fallback registers in package.postload behind an appended searcher, not
	// in package.preload.
	assert.Contains(t, output, "package.postload[\"main\"] = function(...)", "fallback should register in package.postload")
	assert.Contains(t, output, "searchers[#searchers+1] = function(mod)", "fallback should append a searcher")
	assert.NotContains(t, output, "package.preload[", "fallback should not use package.preload")
}
