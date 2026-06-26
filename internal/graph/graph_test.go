package graph

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bigbes/lua-amalgamate/internal/config"
	"github.com/bigbes/lua-amalgamate/internal/parse"
	"github.com/bigbes/lua-amalgamate/internal/resolve"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGraphLinearDependency(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Lua files: main.lua -> a.lua -> b.lua
	mainContent := `require("a")`
	aContent := `require("b")`
	bContent := `print("b")`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.lua"), []byte(mainContent), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "a.lua"), []byte(aContent), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "b.lua"), []byte(bContent), 0644))

	cfg := config.Config{
		Entry:  filepath.Join(tmpDir, "main.lua"),
		Root:   tmpDir,
		Path:   "?.lua",
		Strict: true,
	}

	parser := parse.New()
	resolver := resolve.New(cfg.Root, nil, cfg.Path)
	g, err := Build(&cfg, parser, resolver)
	require.NoError(t, err, "Build() error")

	require.NotNil(t, g.Entry, "Build() Entry is nil")
	assert.Equal(t, filepath.Join(tmpDir, "main.lua"), g.Entry.FilePath, "Entry FilePath")

	// Should have 3 modules
	assert.Len(t, g.Modules, 3, "len(g.Modules)")

	// Check module names
	modulesByPath := make(map[string]*Module)
	for _, m := range g.Modules {
		modulesByPath[m.FilePath] = m
	}

	mainMod := modulesByPath[filepath.Join(tmpDir, "main.lua")]
	aMod := modulesByPath[filepath.Join(tmpDir, "a.lua")]
	bMod := modulesByPath[filepath.Join(tmpDir, "b.lua")]

	require.NotNil(t, mainMod, "Some modules missing")
	require.NotNil(t, aMod, "Some modules missing")
	require.NotNil(t, bMod, "Some modules missing")

	// Check requires
	require.Len(t, mainMod.Requires, 1, "main.Requires")
	assert.Equal(t, "a", mainMod.Requires[0].Name, "main.Requires[0].Name")
	require.Len(t, aMod.Requires, 1, "a.Requires")
	assert.Equal(t, "b", aMod.Requires[0].Name, "a.Requires[0].Name")
	assert.Len(t, bMod.Requires, 0, "b.Requires")
}

func TestGraphCircularDependency(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Lua files: a.lua -> b.lua -> a.lua (circular)
	aContent := `require("b")`
	bContent := `require("a")`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "a.lua"), []byte(aContent), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "b.lua"), []byte(bContent), 0644))

	cfg := config.Config{
		Entry:  filepath.Join(tmpDir, "a.lua"),
		Root:   tmpDir,
		Path:   "?.lua",
		Strict: true,
	}

	parser := parse.New()
	resolver := resolve.New(cfg.Root, nil, cfg.Path)
	g, err := Build(&cfg, parser, resolver)
	require.NoError(t, err, "Build() error")

	// Should have 2 modules (circular is fine)
	assert.Len(t, g.Modules, 2, "len(g.Modules)")
}

func TestGraphDynamicRequireWarning(t *testing.T) {
	tmpDir := t.TempDir()

	content := `local mod = "a"; require(mod)`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.lua"), []byte(content), 0644))

	cfg := config.Config{
		Entry:  filepath.Join(tmpDir, "main.lua"),
		Root:   tmpDir,
		Path:   "?.lua",
		Strict: false, // not strict, should produce warning
	}

	parser := parse.New()
	resolver := resolve.New(cfg.Root, nil, cfg.Path)
	g, err := Build(&cfg, parser, resolver)
	require.NoError(t, err, "Build() error")

	// Should have warning about dynamic require
	assert.Len(t, g.Warnings, 1, "len(g.Warnings)")
}

func TestGraphStrictModeError(t *testing.T) {
	tmpDir := t.TempDir()

	content := `require("nonexistent")`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.lua"), []byte(content), 0644))

	cfg := config.Config{
		Entry:  filepath.Join(tmpDir, "main.lua"),
		Root:   tmpDir,
		Path:   "?.lua",
		Strict: true, // strict mode, should error
	}

	parser := parse.New()
	resolver := resolve.New(cfg.Root, nil, cfg.Path)
	_, err := Build(&cfg, parser, resolver)
	require.Error(t, err, "Build() expected error for unresolved require in strict mode")
}

func TestGraphPackagePrefix(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Lua files: main.lua -> a.lua
	mainContent := `require("a")`
	aContent := `return {}`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.lua"), []byte(mainContent), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "a.lua"), []byte(aContent), 0644))

	cfg := config.Config{
		Entry:         filepath.Join(tmpDir, "main.lua"),
		Root:          tmpDir,
		Path:          "?.lua",
		Strict:        true,
		PackagePrefix: "mypkg",
	}

	parser := parse.New()
	resolver := resolve.New(cfg.Root, nil, cfg.Path)
	g, err := Build(&cfg, parser, resolver)
	require.NoError(t, err, "Build() error")

	// Should have 2 modules
	assert.Len(t, g.Modules, 2, "len(g.Modules)")

	// Check module names
	for _, mod := range g.Modules {
		// Each module should have at least 2 names (original and prefixed)
		assert.GreaterOrEqual(t, len(mod.Names), 2, "module %v has too few names", mod.FilePath)
		// First name for entry should be prefixed
		if mod == g.Entry {
			assert.True(t, strings.HasPrefix(mod.Names[0], "mypkg."), "entry module first name = %q, want prefixed with 'mypkg.'", mod.Names[0])
		}
		// Check that both original and prefixed names exist
		hasOriginal := false
		hasPrefixed := false
		for _, name := range mod.Names {
			if name == "main" || name == "a" {
				hasOriginal = true
			}
			if strings.HasPrefix(name, "mypkg.") {
				hasPrefixed = true
			}
		}
		assert.True(t, hasOriginal, "module missing original name")
		assert.True(t, hasPrefixed, "module missing prefixed name")
	}
}

func TestGraphSkipPackages(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Lua files: main.lua -> a.lua, but a.lua is skipped
	mainContent := `require("a")`
	aContent := `return {}`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.lua"), []byte(mainContent), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "a.lua"), []byte(aContent), 0644))

	cfg := config.Config{
		Entry:        filepath.Join(tmpDir, "main.lua"),
		Root:         tmpDir,
		Path:         "?.lua",
		Strict:       true,
		SkipPackages: []string{"a"},
	}

	parser := parse.New()
	resolver := resolve.New(cfg.Root, nil, cfg.Path)
	g, err := Build(&cfg, parser, resolver)
	require.NoError(t, err, "Build() error")

	// Should have only 1 module (main)
	require.Len(t, g.Modules, 1, "len(g.Modules)")
	assert.Same(t, g.Entry, g.Modules[0], "only module should be entry")
	// Should have a warning about skipped package
	foundSkipWarning := false
	for _, w := range g.Warnings {
		if strings.Contains(w.Message, "skipped package") {
			foundSkipWarning = true
			break
		}
	}
	assert.True(t, foundSkipWarning, "expected skip warning not found")
}

func TestGraphIncludePackages(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Lua files: main.lua (no requires), a.lua, b.lua
	mainContent := `print("main")`
	aContent := `require("b")`
	bContent := `print("b")`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.lua"), []byte(mainContent), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "a.lua"), []byte(aContent), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "b.lua"), []byte(bContent), 0644))

	cfg := config.Config{
		Entry:           filepath.Join(tmpDir, "main.lua"),
		Root:            tmpDir,
		Path:            "?.lua",
		Strict:          true,
		IncludePackages: []string{"a"},
	}

	parser := parse.New()
	resolver := resolve.New(cfg.Root, nil, cfg.Path)
	g, err := Build(&cfg, parser, resolver)
	require.NoError(t, err, "Build() error")

	// Should have 3 modules (main, a, b) because a includes b
	assert.Len(t, g.Modules, 3, "len(g.Modules)")
	// Check that a and b are present
	foundA := false
	foundB := false
	for _, mod := range g.Modules {
		base := filepath.Base(mod.FilePath)
		if base == "a.lua" {
			foundA = true
		}
		if base == "b.lua" {
			foundB = true
		}
	}
	assert.True(t, foundA, "module a.lua not found in graph")
	assert.True(t, foundB, "module b.lua not found in graph")
}

func TestGraphIncludeSkipConflict(t *testing.T) {
	tmpDir := t.TempDir()

	mainContent := `print("main")`
	aContent := `print("a")`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.lua"), []byte(mainContent), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "a.lua"), []byte(aContent), 0644))

	cfg := config.Config{
		Entry:           filepath.Join(tmpDir, "main.lua"),
		Root:            tmpDir,
		Path:            "?.lua",
		Strict:          true,
		IncludePackages: []string{"a"},
		SkipPackages:    []string{"a"},
	}

	parser := parse.New()
	resolver := resolve.New(cfg.Root, nil, cfg.Path)
	g, err := Build(&cfg, parser, resolver)
	require.NoError(t, err, "Build() error")

	// Should have only 1 module (main) because skip takes precedence
	assert.Len(t, g.Modules, 1, "len(g.Modules)")
	// Should have warning about skipped include
	foundSkipWarning := false
	for _, w := range g.Warnings {
		if strings.Contains(w.Message, "include package") && strings.Contains(w.Message, "skipped") {
			foundSkipWarning = true
			break
		}
	}
	assert.True(t, foundSkipWarning, "expected skip warning for include package not found")
}
