package main

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
)

// binPath is the freshly built CLI binary, shared across all CLI tests.
var binPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "lua-amalgamate-bin-*")
	if err != nil {
		panic(err)
	}
	binPath = filepath.Join(dir, "lua-amalgamate")
	if out, err := exec.Command("go", "build", "-o", binPath, ".").CombinedOutput(); err != nil {
		os.Stderr.Write(out)
		os.RemoveAll(dir)
		panic("build failed: " + err.Error())
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// runCLI runs the built binary and returns stdout, stderr, and the exit code.
func runCLI(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			code = ee.ExitCode()
		} else {
			t.Fatalf("run %v: %v", args, err)
		}
	}
	return stdout.String(), stderr.String(), code
}

// writeProject creates a minimal two-module project and returns its directory.
// main.lua requires "module"; module.lua carries a comment so transform flags
// have something observable to remove.
func writeProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.lua"),
		[]byte("local m = require(\"module\")\nprint(m.greet())\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "module.lua"),
		[]byte("-- UNIQUE_COMMENT_MARKER\nlocal M = {}\nfunction M.greet() return \"hi\" end\nreturn M\n"), 0o644))
	return dir
}

func base(dir string) []string {
	return []string{"--entry", filepath.Join(dir, "main.lua"), "--root", dir, "--output", "-"}
}

func TestCLI_BasicStdout(t *testing.T) {
	dir := writeProject(t)
	out, _, code := runCLI(t, base(dir)...)
	assert.Equal(t, 0, code)
	assert.Contains(t, out, `package.preload["main"]`)
	assert.Contains(t, out, `package.preload["module"]`)
	assert.Contains(t, out, `return require("main")`)
}

func TestCLI_Debug(t *testing.T) {
	dir := writeProject(t)
	plain, _, _ := runCLI(t, base(dir)...)
	dbg, _, code := runCLI(t, append(base(dir), "--debug")...)
	assert.Equal(t, 0, code)
	assert.NotContains(t, plain, "loadstring or load", "plain output should not use load()")
	assert.Contains(t, dbg, "(loadstring or load)", "--debug should load() module bodies")
	assert.Contains(t, dbg, "module.lua", "--debug chunk name should reference the source file")
}

func TestCLI_Fallback(t *testing.T) {
	dir := writeProject(t)
	out, _, code := runCLI(t, append(base(dir), "--fallback")...)
	assert.Equal(t, 0, code)
	assert.Contains(t, out, "package.postload[")
	assert.Contains(t, out, "searchers[#searchers+1] = function(mod)")
	assert.NotContains(t, out, "package.preload[")
}

func TestCLI_Shebang(t *testing.T) {
	dir := writeProject(t)
	out, _, code := runCLI(t, append(base(dir), "--shebang", "#!/usr/bin/env lua")...)
	assert.Equal(t, 0, code)
	assert.True(t, strings.HasPrefix(out, "#!/usr/bin/env lua\n"), "shebang must be first line")
}

func TestCLI_NoArgFix(t *testing.T) {
	dir := writeProject(t)
	withFix, _, _ := runCLI(t, base(dir)...)
	without, _, code := runCLI(t, append(base(dir), "--no-arg-fix")...)
	assert.Equal(t, 0, code)
	assert.Contains(t, withFix, "local arg = _G.arg", "arg alias present by default")
	assert.NotContains(t, without, "local arg = _G.arg", "--no-arg-fix omits the arg alias")
}

func TestCLI_Skip(t *testing.T) {
	dir := writeProject(t)
	out, stderr, code := runCLI(t, append(base(dir), "--skip", "module")...)
	assert.Equal(t, 0, code)
	assert.NotContains(t, out, `package.preload["module"]`, "--skip should exclude the module")
	assert.Contains(t, stderr, "skipped package")
}

func TestCLI_Include(t *testing.T) {
	dir := writeProject(t)
	// An orphan module not required by anything.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "orphan.lua"), []byte("return 42\n"), 0o644))

	without, _, _ := runCLI(t, base(dir)...)
	with, _, code := runCLI(t, append(base(dir), "--include", "orphan")...)
	assert.Equal(t, 0, code)
	assert.NotContains(t, without, `package.preload["orphan"]`, "orphan absent without --include")
	assert.Contains(t, with, `package.preload["orphan"]`, "--include should force-bundle the orphan")
}

func TestCLI_Strict(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.lua"),
		[]byte("require(\"does_not_exist\")\n"), 0o644))
	entry := []string{"--entry", filepath.Join(dir, "main.lua"), "--root", dir, "--output", "-"}

	_, _, codeLoose := runCLI(t, entry...)
	assert.Equal(t, 0, codeLoose, "without --strict an unresolved require is a warning")

	_, stderr, codeStrict := runCLI(t, append(entry, "--strict")...)
	assert.NotEqual(t, 0, codeStrict, "--strict should fail on an unresolved require")
	assert.Contains(t, stderr, "unresolved")
}

func TestCLI_RemoveComments(t *testing.T) {
	dir := writeProject(t)
	plain, _, _ := runCLI(t, base(dir)...)
	stripped, _, code := runCLI(t, append(base(dir), "--remove-comments")...)
	assert.Equal(t, 0, code)
	assert.Contains(t, plain, "UNIQUE_COMMENT_MARKER", "comment present by default")
	assert.NotContains(t, stripped, "UNIQUE_COMMENT_MARKER", "--remove-comments should strip it")
}

func TestCLI_PrefixSuffix(t *testing.T) {
	dir := writeProject(t)
	out, _, code := runCLI(t, append(base(dir),
		"--prefix", "print('PREFIX_MARK')", "--suffix", "print('SUFFIX_MARK')")...)
	assert.Equal(t, 0, code)
	pre := strings.Index(out, "PREFIX_MARK")
	req := strings.Index(out, `return require("main")`)
	suf := strings.Index(out, "SUFFIX_MARK")
	require.True(t, pre >= 0 && req >= 0 && suf >= 0, "all markers present")
	assert.Less(t, pre, req, "prefix before the entry require")
	assert.Less(t, req, suf, "suffix after the entry require")
}

func TestCLI_PackagePrefix(t *testing.T) {
	dir := writeProject(t)
	out, _, code := runCLI(t, append(base(dir), "--package-prefix", "mypkg")...)
	assert.Equal(t, 0, code)
	assert.Contains(t, out, `package.preload["mypkg.main"]`, "--package-prefix should namespace module names")
	assert.Contains(t, out, `return require("mypkg.main")`)
}

func TestCLI_PackageName(t *testing.T) {
	// Modules named mylib.* but files live flat in dir.
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.lua"),
		[]byte("local u = require(\"mylib.util\")\nprint(u)\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "util.lua"), []byte("return 1\n"), 0o644))
	entry := []string{"--entry", filepath.Join(dir, "main.lua"), "--root", dir, "--output", "-"}

	out, _, code := runCLI(t, append(entry, "--package-name", "mylib")...)
	assert.Equal(t, 0, code)
	// Strips mylib. to resolve util.lua, then re-adds it to the output name.
	assert.Contains(t, out, `package.preload["mylib.util"]`, "--package-name should resolve flat and re-add the prefix")
}

func TestCLI_StripPrefix(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.lua"),
		[]byte("local u = require(\"mylib.util\")\nprint(u)\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "util.lua"), []byte("return 1\n"), 0o644))
	entry := []string{"--entry", filepath.Join(dir, "main.lua"), "--root", dir, "--output", "-"}

	// strip-prefix changes resolution (mylib.util -> util.lua) but keeps the
	// requested name. Without it the require can't resolve, so the module is
	// absent; with it the module is bundled under its requested name.
	without, _, _ := runCLI(t, entry...)
	assert.NotContains(t, without, `package.preload["mylib.util"]`, "mylib.util shouldn't resolve without strip-prefix")

	with, _, code := runCLI(t, append(entry, "--strip-prefix", "mylib")...)
	assert.Equal(t, 0, code)
	assert.Contains(t, with, `package.preload["mylib.util"]`, "--strip-prefix should resolve mylib.util to util.lua and bundle it")
}

func TestCLI_OutputToFile(t *testing.T) {
	dir := writeProject(t)
	outFile := filepath.Join(t.TempDir(), "bundle.lua")
	_, _, code := runCLI(t, "--entry", filepath.Join(dir, "main.lua"), "--root", dir, "--output", outFile)
	assert.Equal(t, 0, code)
	data, err := os.ReadFile(outFile)
	require.NoError(t, err)
	assert.Contains(t, string(data), `return require("main")`)
}
