package emit

import (
	"bytes"
	"strings"
	"testing"

	"github.com/bigbes/lua-amalgamate/internal/graph"
	"github.com/bigbes/lua-amalgamate/internal/parse"
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
	if err := Emit(&buf, g, nil, "", ""); err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

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
		if !strings.Contains(output, line) {
			t.Errorf("Output missing line %q", line)
		}
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
	if err := Emit(&buf, g, nil, "", ""); err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	output := buf.String()
	// Should contain both modules
	if !strings.Contains(output, "package.preload[\"main\"]") {
		t.Error("Output missing main module")
	}
	if !strings.Contains(output, "package.preload[\"a\"]") {
		t.Error("Output missing a module")
	}
	if !strings.Contains(output, "require(\"main\")") {
		t.Error("Output missing entry require")
	}
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
	if err := Emit(&buf, g, nil, "", ""); err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	output := buf.String()
	// Primary name carries the real loader; aliases delegate to it so the
	// module body runs exactly once regardless of which name is required.
	if !strings.Contains(output, "package.preload[\"foo\"] = function(...)") {
		t.Error("Output missing primary loader for foo")
	}
	if !strings.Contains(output, "package.preload[\"foo.bar\"] = function(...) return require(\"foo\") end") {
		t.Error("Output missing delegating alias foo.bar")
	}
	if !strings.Contains(output, "package.preload[\"foo/bar\"] = function(...) return require(\"foo\") end") {
		t.Error("Output missing delegating alias foo/bar")
	}
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
	if err := Emit(&buf, g, nil, "", suffixCode); err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "print('suffix')") {
		t.Error("Output missing suffix code")
	}
	// Ensure suffix appears after require
	requireIdx := strings.Index(output, "require(\"main\")")
	suffixIdx := strings.Index(output, "print('suffix')")
	if requireIdx == -1 || suffixIdx == -1 || suffixIdx < requireIdx {
		t.Error("Suffix code should appear after require")
	}
}
