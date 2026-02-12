package integration

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bigbes/lua-amalgamator/internal/config"
	"github.com/bigbes/lua-amalgamator/internal/emit"
	"github.com/bigbes/lua-amalgamator/internal/graph"
	"github.com/bigbes/lua-amalgamator/internal/parse"
	"github.com/bigbes/lua-amalgamator/internal/resolve"
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
	if err := emit.Emit(&buf, g, nil, "", ""); err != nil {
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
