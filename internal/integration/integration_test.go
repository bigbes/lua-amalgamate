package integration

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	require.NoError(t, err)

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
	require.NoError(t, cfg.ResolveRoot(), "resolve root")

	parser := parse.New()
	resolver := resolve.New(cfg.Root, nil, cfg.Path)
	g, err := graph.Build(&cfg, parser, resolver)
	require.NoError(t, err, "build graph")

	var buf bytes.Buffer
	require.NoError(t, emit.Emit(&buf, g, nil, emit.Options{}), "emit")
	bundle := buf.Bytes()

	// Write bundle to temp file
	tmpFile, err := os.CreateTemp("", "amalg-test-*.lua")
	require.NoError(t, err, "create temp file")
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write(bundle)
	require.NoError(t, err, "write temp file")
	require.NoError(t, tmpFile.Close(), "close temp file")

	// Run bundle
	bundleOut, bundleErr := runLua(t, luaPath, tmpFile.Name(), dir)

	// Compare outputs
	assert.Equal(t, origOut, bundleOut, "output mismatch\noriginal output:\n%s\nbundle output:\n%s", origOut, bundleOut)
	// Compare errors: both empty or both non-empty
	assert.Equal(t, origErr == "", bundleErr == "", "error presence mismatch\noriginal error:\n%s\nbundle error:\n%s", origErr, bundleErr)
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
	require.NoError(t, os.WriteFile(mainPath, []byte("local m = require(\"boom\")\nm.go()\n"), 0o644))
	// error() is on line 3 of boom.lua.
	require.NoError(t, os.WriteFile(boomPath, []byte("local M = {}\nfunction M.go()\n  error(\"kaboom\")\nend\nreturn M\n"), 0o644))

	cfg := config.Config{Entry: mainPath, Root: dir, Path: "?.lua;?/init.lua"}
	require.NoError(t, cfg.ResolveRoot(), "resolve root")
	g, err := graph.Build(&cfg, parse.New(), resolve.New(cfg.Root, nil, cfg.Path))
	require.NoError(t, err, "build graph")

	var buf bytes.Buffer
	require.NoError(t, emit.Emit(&buf, g, nil, emit.Options{Debug: true}), "emit")

	bundlePath := filepath.Join(dir, "bundle.lua")
	require.NoError(t, os.WriteFile(bundlePath, buf.Bytes(), 0o644))

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
	assert.Contains(t, errLine, want, "debug error should point at original file:line\nwant substring: %q\ngot first line: %q\nfull stderr:\n%s", want, errLine, stderr)
	assert.NotContains(t, errLine, "bundle.lua:", "debug error should not originate from bundle.lua\ngot first line: %q", errLine)
}

// TestFallbackOnDiskOverride verifies that in fallback mode an on-disk module
// found on package.path takes precedence over the embedded copy, while a normal
// bundle uses the embedded copy.
func TestFallbackOnDiskOverride(t *testing.T) {
	luaPath := findLua()
	if luaPath == "" {
		t.Skip("lua interpreter not found in PATH")
	}

	src := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(src, "main.lua"), []byte("local g = require(\"greet\")\nprint(g())\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "greet.lua"), []byte("return function() return \"EMBEDDED\" end\n"), 0o644))

	cfg := config.Config{Entry: filepath.Join(src, "main.lua"), Root: src, Path: "?.lua;?/init.lua"}
	require.NoError(t, cfg.ResolveRoot())
	g, err := graph.Build(&cfg, parse.New(), resolve.New(cfg.Root, nil, cfg.Path))
	require.NoError(t, err, "build graph")

	emitBundle := func(opts emit.Options) string {
		var buf bytes.Buffer
		require.NoError(t, emit.Emit(&buf, g, nil, opts), "emit")
		return buf.String()
	}

	// Run directory holds an on-disk override of greet reachable via ./?.lua.
	run := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(run, "greet.lua"), []byte("return function() return \"ON-DISK\" end\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(run, "normal.lua"), []byte(emitBundle(emit.Options{})), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(run, "fallback.lua"), []byte(emitBundle(emit.Options{Fallback: true})), 0o644))

	runWithPath := func(script string) string {
		cmd := exec.Command(luaPath, script)
		cmd.Dir = run
		cmd.Env = append(os.Environ(), "LUA_PATH=./?.lua;;")
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		require.NoError(t, cmd.Run(), "run %s: %s", script, stderr.String())
		return strings.TrimSpace(stdout.String())
	}

	assert.Equal(t, "EMBEDDED", runWithPath("normal.lua"), "normal bundle should use the embedded module")
	assert.Equal(t, "ON-DISK", runWithPath("fallback.lua"), "fallback bundle should prefer the on-disk module")
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
