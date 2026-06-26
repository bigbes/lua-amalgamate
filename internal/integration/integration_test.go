package integration

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bigbes/lua-amalgamate/internal/config"
	"github.com/bigbes/lua-amalgamate/internal/emit"
	"github.com/bigbes/lua-amalgamate/internal/graph"
	"github.com/bigbes/lua-amalgamate/internal/parse"
	"github.com/bigbes/lua-amalgamate/internal/resolve"
)

func findLua() string {
	if path, err := exec.LookPath("lua"); err == nil {
		return path
	}
	if path, err := exec.LookPath("lua5.1"); err == nil {
		return path
	}
	if path, err := exec.LookPath("lua5.2"); err == nil {
		return path
	}
	if path, err := exec.LookPath("lua5.3"); err == nil {
		return path
	}
	if path, err := exec.LookPath("lua5.4"); err == nil {
		return path
	}
	return ""
}

func TestIntegration(t *testing.T) {
	luaPath := findLua()
	if luaPath == "" {
		t.Skip("lua interpreter not found in PATH")
	}

	testDirs, err := filepath.Glob("../../testdata/*")
	if err != nil {
		t.Fatal(err)
	}

	skipDirs := map[string]bool{
		"circular": true, // original fails with C stack overflow; amalgamator fixes circular deps
	}

	for _, dir := range testDirs {
		// Skip directories that don't have a main.lua
		mainPath := filepath.Join(dir, "main.lua")
		if _, err := os.Stat(mainPath); err != nil {
			continue
		}
		name := filepath.Base(dir)
		if skipDirs[name] {
			continue
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			runIntegrationTest(t, luaPath, dir, mainPath)
		})
	}
}

func runIntegrationTest(t *testing.T, luaPath, dir, entryPath string) {
	// Run original project
	origOut, origErr := runLua(t, luaPath, entryPath, dir)

	// Build amalgamated bundle
	cfg := config.Config{
		Entry:  entryPath,
		Root:   dir,
		Path:   "?.lua;?/init.lua",
		Strict: false,
	}
	if err := cfg.ResolveRoot(); err != nil {
		t.Fatalf("resolve root: %v", err)
	}

	parser := parse.New()
	resolver := resolve.New(cfg.Root, nil, cfg.Path)
	g, err := graph.Build(&cfg, parser, resolver)
	if err != nil {
		t.Fatalf("build graph: %v", err)
	}

	var buf bytes.Buffer
	if err := emit.Emit(&buf, g, nil, emit.Options{}); err != nil {
		t.Fatalf("emit: %v", err)
	}
	bundle := buf.Bytes()

	// Write bundle to temp file
	tmpFile, err := os.CreateTemp("", "amalg-test-*.lua")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(bundle); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("close temp file: %v", err)
	}

	// Run bundle
	bundleOut, bundleErr := runLua(t, luaPath, tmpFile.Name(), dir)

	// Compare outputs
	if origOut != bundleOut {
		t.Errorf("output mismatch\noriginal output:\n%s\nbundle output:\n%s", origOut, bundleOut)
	}
	// Compare errors: both empty or both non-empty
	if (origErr == "") != (bundleErr == "") {
		t.Errorf("error presence mismatch\noriginal error:\n%s\nbundle error:\n%s", origErr, bundleErr)
	}
}

// TestDebugTraceback verifies that --debug makes runtime errors report the
// original module file and line number instead of an offset into the bundle.
func TestDebugTraceback(t *testing.T) {
	luaPath := findLua()
	if luaPath == "" {
		t.Skip("lua interpreter not found in PATH")
	}

	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.lua")
	boomPath := filepath.Join(dir, "boom.lua")
	if err := os.WriteFile(mainPath, []byte("local m = require(\"boom\")\nm.go()\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// error() is on line 3 of boom.lua.
	if err := os.WriteFile(boomPath, []byte("local M = {}\nfunction M.go()\n  error(\"kaboom\")\nend\nreturn M\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{Entry: mainPath, Root: dir, Path: "?.lua;?/init.lua"}
	if err := cfg.ResolveRoot(); err != nil {
		t.Fatalf("resolve root: %v", err)
	}
	g, err := graph.Build(&cfg, parse.New(), resolve.New(cfg.Root, nil, cfg.Path))
	if err != nil {
		t.Fatalf("build graph: %v", err)
	}

	var buf bytes.Buffer
	if err := emit.Emit(&buf, g, nil, emit.Options{Debug: true}); err != nil {
		t.Fatalf("emit: %v", err)
	}

	bundlePath := filepath.Join(dir, "bundle.lua")
	if err := os.WriteFile(bundlePath, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	_, stderr := runLua(t, luaPath, bundlePath, dir)
	// The error origin (first line) must report the original module file:line,
	// not an offset into the bundle.
	errLine := stderr
	if i := strings.IndexByte(stderr, '\n'); i >= 0 {
		errLine = stderr[:i]
	}
	// Lua truncates long chunk names (LUA_IDSIZE) with a leading "...", so match
	// on the base name + line rather than the full temp path.
	want := "boom.lua:3"
	if !strings.Contains(errLine, want) {
		t.Errorf("debug error should point at original file:line\nwant substring: %q\ngot first line: %q\nfull stderr:\n%s", want, errLine, stderr)
	}
	if strings.Contains(errLine, "bundle.lua:") {
		t.Errorf("debug error should not originate from bundle.lua\ngot first line: %q", errLine)
	}
}

func runLua(t *testing.T, luaPath, scriptPath, workingDir string) (string, string) {
	cmd := exec.Command(luaPath, scriptPath)
	cmd.Dir = workingDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// Ignore error; we'll capture stderr
	}
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String())
}
