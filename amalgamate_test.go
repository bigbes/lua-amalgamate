package amalgamate_test

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	amalgamate "github.com/bigbes/lua-amalgamate"
)

// writeProject creates a minimal main + module project and returns its dir.
func writeProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.lua"),
		[]byte("local m = require(\"module\")\nprint(m.greet())\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "module.lua"),
		[]byte("local M = {}\nfunction M.greet() return \"hi\" end\nreturn M\n"), 0o644))
	return dir
}

func TestDefaultOptions(t *testing.T) {
	opts := amalgamate.DefaultOptions()
	assert.True(t, opts.ArgFix, "arg fix on by default")
	assert.Equal(t, "?.lua;?/init.lua", opts.Path)
	assert.False(t, opts.Debug)
}

func TestBundle(t *testing.T) {
	dir := writeProject(t)
	opts := amalgamate.DefaultOptions()
	opts.Entry = filepath.Join(dir, "main.lua")
	opts.Root = dir

	var buf bytes.Buffer
	res, err := amalgamate.Bundle(opts, &buf)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Empty(t, res.Warnings)

	out := buf.String()
	assert.Contains(t, out, `package.preload["main"]`)
	assert.Contains(t, out, `package.preload["module"]`)
	assert.Contains(t, out, `return require("main")`)
}

func TestBundleOptionsThread(t *testing.T) {
	dir := writeProject(t)
	opts := amalgamate.DefaultOptions()
	opts.Entry = filepath.Join(dir, "main.lua")
	opts.Root = dir
	opts.Debug = true
	opts.Shebang = "#!/usr/bin/env lua"

	var buf bytes.Buffer
	_, err := amalgamate.Bundle(opts, &buf)
	require.NoError(t, err)

	out := buf.String()
	assert.True(t, strings.HasPrefix(out, "#!/usr/bin/env lua\n"), "shebang threaded through")
	assert.Contains(t, out, "(loadstring or load)", "debug threaded through")
}

func TestBundleWarnings(t *testing.T) {
	dir := t.TempDir()
	// A dynamic require can't be resolved statically -> warning, not error.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.lua"),
		[]byte("local n = \"mod\"\nrequire(n)\n"), 0o644))

	opts := amalgamate.DefaultOptions()
	opts.Entry = filepath.Join(dir, "main.lua")
	opts.Root = dir

	var buf bytes.Buffer
	res, err := amalgamate.Bundle(opts, &buf)
	require.NoError(t, err)
	require.Len(t, res.Warnings, 1, "dynamic require should produce a warning")
	// Warnings are machine-readable, not just strings.
	assert.Equal(t, amalgamate.WarnDynamicRequire, res.Warnings[0].Kind)
}

func TestBundleUnresolvedWarningKind(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.lua"),
		[]byte("require(\"nope\")\n"), 0o644))

	opts := amalgamate.DefaultOptions()
	opts.Entry = filepath.Join(dir, "main.lua")
	opts.Root = dir

	res, err := amalgamate.Bundle(opts, &bytes.Buffer{})
	require.NoError(t, err)
	require.Len(t, res.Warnings, 1)
	assert.Equal(t, amalgamate.WarnUnresolved, res.Warnings[0].Kind)
	assert.Equal(t, "nope", res.Warnings[0].Module)
}

func TestBundleStrictError(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.lua"),
		[]byte("require(\"does_not_exist\")\n"), 0o644))

	opts := amalgamate.DefaultOptions()
	opts.Entry = filepath.Join(dir, "main.lua")
	opts.Root = dir
	opts.Strict = true

	var buf bytes.Buffer
	_, err := amalgamate.Bundle(opts, &buf)
	require.Error(t, err, "strict mode should fail on an unresolved require")

	// The error is classifiable both as a typed error and via its cause.
	var ue *amalgamate.UnresolvedError
	require.ErrorAs(t, err, &ue)
	assert.Equal(t, "does_not_exist", ue.Module)
	assert.ErrorIs(t, err, amalgamate.ErrModuleNotFound)
}

func TestBundleMissingEntry(t *testing.T) {
	_, err := amalgamate.Bundle(amalgamate.DefaultOptions(), &bytes.Buffer{})
	require.ErrorIs(t, err, amalgamate.ErrNoEntry)
	// Sanity: errors.Is works against the exported sentinel.
	assert.True(t, errors.Is(err, amalgamate.ErrNoEntry))
}

// TestBundleRunsUnderLua proves the public API produces a runnable bundle.
func TestBundleRunsUnderLua(t *testing.T) {
	lua, err := exec.LookPath("lua")
	if err != nil {
		t.Skip("lua interpreter not found in PATH")
	}
	dir := writeProject(t)
	opts := amalgamate.DefaultOptions()
	opts.Entry = filepath.Join(dir, "main.lua")
	opts.Root = dir

	var buf bytes.Buffer
	_, err = amalgamate.Bundle(opts, &buf)
	require.NoError(t, err)

	bundle := filepath.Join(t.TempDir(), "bundle.lua")
	require.NoError(t, os.WriteFile(bundle, buf.Bytes(), 0o644))

	out, err := exec.Command(lua, bundle).CombinedOutput()
	require.NoError(t, err, "bundle should run: %s", out)
	assert.Equal(t, "hi", strings.TrimSpace(string(out)))
}
