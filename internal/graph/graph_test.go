package graph

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bigbes/lua-amalgamator/internal/config"
	"github.com/bigbes/lua-amalgamator/internal/parse"
	"github.com/bigbes/lua-amalgamator/internal/resolve"
)

func TestGraphLinearDependency(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Lua files: main.lua -> a.lua -> b.lua
	mainContent := `require("a")`
	aContent := `require("b")`
	bContent := `print("b")`

	if err := os.WriteFile(filepath.Join(tmpDir, "main.lua"), []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "a.lua"), []byte(aContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "b.lua"), []byte(bContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		Entry:  filepath.Join(tmpDir, "main.lua"),
		Root:   tmpDir,
		Path:   "?.lua",
		Strict: true,
	}

	parser := parse.New()
	resolver := resolve.New(cfg.Root, nil, cfg.Path)
	g, err := Build(&cfg, parser, resolver)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if g.Entry == nil {
		t.Fatal("Build() Entry is nil")
	}
	if g.Entry.FilePath != filepath.Join(tmpDir, "main.lua") {
		t.Errorf("Entry FilePath = %v, want %v", g.Entry.FilePath, filepath.Join(tmpDir, "main.lua"))
	}

	// Should have 3 modules
	if len(g.Modules) != 3 {
		t.Errorf("len(g.Modules) = %d, want 3", len(g.Modules))
	}

	// Check module names
	modulesByPath := make(map[string]*Module)
	for _, m := range g.Modules {
		modulesByPath[m.FilePath] = m
	}

	mainMod := modulesByPath[filepath.Join(tmpDir, "main.lua")]
	aMod := modulesByPath[filepath.Join(tmpDir, "a.lua")]
	bMod := modulesByPath[filepath.Join(tmpDir, "b.lua")]

	if mainMod == nil || aMod == nil || bMod == nil {
		t.Fatal("Some modules missing")
	}

	// Check requires
	if len(mainMod.Requires) != 1 || mainMod.Requires[0].Name != "a" {
		t.Errorf("main.Requires = %v, want [{a}]", mainMod.Requires)
	}
	if len(aMod.Requires) != 1 || aMod.Requires[0].Name != "b" {
		t.Errorf("a.Requires = %v, want [{b}]", aMod.Requires)
	}
	if len(bMod.Requires) != 0 {
		t.Errorf("b.Requires = %v, want []", bMod.Requires)
	}
}

func TestGraphCircularDependency(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Lua files: a.lua -> b.lua -> a.lua (circular)
	aContent := `require("b")`
	bContent := `require("a")`

	if err := os.WriteFile(filepath.Join(tmpDir, "a.lua"), []byte(aContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "b.lua"), []byte(bContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		Entry:  filepath.Join(tmpDir, "a.lua"),
		Root:   tmpDir,
		Path:   "?.lua",
		Strict: true,
	}

	parser := parse.New()
	resolver := resolve.New(cfg.Root, nil, cfg.Path)
	g, err := Build(&cfg, parser, resolver)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	// Should have 2 modules (circular is fine)
	if len(g.Modules) != 2 {
		t.Errorf("len(g.Modules) = %d, want 2", len(g.Modules))
	}
}

func TestGraphDynamicRequireWarning(t *testing.T) {
	tmpDir := t.TempDir()

	content := `local mod = "a"; require(mod)`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.lua"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		Entry:  filepath.Join(tmpDir, "main.lua"),
		Root:   tmpDir,
		Path:   "?.lua",
		Strict: false, // not strict, should produce warning
	}

	parser := parse.New()
	resolver := resolve.New(cfg.Root, nil, cfg.Path)
	g, err := Build(&cfg, parser, resolver)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	// Should have warning about dynamic require
	if len(g.Warnings) != 1 {
		t.Errorf("len(g.Warnings) = %d, want 1", len(g.Warnings))
	}
}

func TestGraphStrictModeError(t *testing.T) {
	tmpDir := t.TempDir()

	content := `require("nonexistent")`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.lua"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		Entry:  filepath.Join(tmpDir, "main.lua"),
		Root:   tmpDir,
		Path:   "?.lua",
		Strict: true, // strict mode, should error
	}

	parser := parse.New()
	resolver := resolve.New(cfg.Root, nil, cfg.Path)
	_, err := Build(&cfg, parser, resolver)
	if err == nil {
		t.Fatal("Build() expected error for unresolved require in strict mode")
	}
}

func TestGraphPackagePrefix(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Lua files: main.lua -> a.lua
	mainContent := `require("a")`
	aContent := `return {}`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.lua"), []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "a.lua"), []byte(aContent), 0644); err != nil {
		t.Fatal(err)
	}

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
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	// Should have 2 modules
	if len(g.Modules) != 2 {
		t.Errorf("len(g.Modules) = %d, want 2", len(g.Modules))
	}

	// Check module names
	for _, mod := range g.Modules {
		// Each module should have at least 2 names (original and prefixed)
		if len(mod.Names) < 2 {
			t.Errorf("module %v has only %d names, want at least 2", mod.FilePath, len(mod.Names))
		}
		// First name for entry should be prefixed
		if mod == g.Entry {
			if !strings.HasPrefix(mod.Names[0], "mypkg.") {
				t.Errorf("entry module first name = %q, want prefixed with 'mypkg.'", mod.Names[0])
			}
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
		if !hasOriginal {
			t.Errorf("module missing original name")
		}
		if !hasPrefixed {
			t.Errorf("module missing prefixed name")
		}
	}
}

func TestGraphSkipPackages(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Lua files: main.lua -> a.lua, but a.lua is skipped
	mainContent := `require("a")`
	aContent := `return {}`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.lua"), []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "a.lua"), []byte(aContent), 0644); err != nil {
		t.Fatal(err)
	}

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
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	// Should have only 1 module (main)
	if len(g.Modules) != 1 {
		t.Errorf("len(g.Modules) = %d, want 1", len(g.Modules))
	}
	if g.Modules[0] != g.Entry {
		t.Error("only module should be entry")
	}
	// Should have a warning about skipped package
	foundSkipWarning := false
	for _, w := range g.Warnings {
		if strings.Contains(w.Message, "skipped package") {
			foundSkipWarning = true
			break
		}
	}
	if !foundSkipWarning {
		t.Error("expected skip warning not found")
	}
}

func TestGraphIncludePackages(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Lua files: main.lua (no requires), a.lua, b.lua
	mainContent := `print("main")`
	aContent := `require("b")`
	bContent := `print("b")`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.lua"), []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "a.lua"), []byte(aContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "b.lua"), []byte(bContent), 0644); err != nil {
		t.Fatal(err)
	}

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
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	// Should have 3 modules (main, a, b) because a includes b
	if len(g.Modules) != 3 {
		t.Errorf("len(g.Modules) = %d, want 3", len(g.Modules))
	}
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
	if !foundA {
		t.Error("module a.lua not found in graph")
	}
	if !foundB {
		t.Error("module b.lua not found in graph")
	}
}

func TestGraphIncludeSkipConflict(t *testing.T) {
	tmpDir := t.TempDir()

	mainContent := `print("main")`
	aContent := `print("a")`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.lua"), []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "a.lua"), []byte(aContent), 0644); err != nil {
		t.Fatal(err)
	}

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
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	// Should have only 1 module (main) because skip takes precedence
	if len(g.Modules) != 1 {
		t.Errorf("len(g.Modules) = %d, want 1", len(g.Modules))
	}
	// Should have warning about skipped include
	foundSkipWarning := false
	for _, w := range g.Warnings {
		if strings.Contains(w.Message, "include package") && strings.Contains(w.Message, "skipped") {
			foundSkipWarning = true
			break
		}
	}
	if !foundSkipWarning {
		t.Error("expected skip warning for include package not found")
	}
}
