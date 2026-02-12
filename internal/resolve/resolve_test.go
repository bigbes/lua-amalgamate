package resolve

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolver(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directory structure
	luaDir := filepath.Join(tmpDir, "lua")
	if err := os.MkdirAll(luaDir, 0755); err != nil {
		t.Fatal(err)
	}
	libDir := filepath.Join(tmpDir, "lib")
	if err := os.MkdirAll(libDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write test Lua files
	fooPath := filepath.Join(luaDir, "foo.lua")
	if err := os.WriteFile(fooPath, []byte("-- foo"), 0644); err != nil {
		t.Fatal(err)
	}
	barInitPath := filepath.Join(libDir, "bar", "init.lua")
	if err := os.MkdirAll(filepath.Dir(barInitPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(barInitPath, []byte("-- bar"), 0644); err != nil {
		t.Fatal(err)
	}

	resolver := New(luaDir, []string{libDir}, "?.lua;?/init.lua")

	tests := []struct {
		name     string
		require  string
		fromDir  string
		wantPath string
		wantName string
	}{
		{
			name:     "simple module",
			require:  "foo",
			fromDir:  luaDir,
			wantPath: fooPath,
			wantName: "foo",
		},
		{
			name:     "init.lua module",
			require:  "bar",
			fromDir:  libDir,
			wantPath: barInitPath,
			wantName: "bar",
		},
		{
			name:     "dot notation",
			require:  "foo.bar",
			fromDir:  luaDir,
			wantPath: "", // not found
			wantName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.Resolve(tt.require, tt.fromDir)
			if tt.wantPath == "" {
				if err == nil {
					t.Errorf("Resolve() expected error, got result %v", result)
				}
				return
			}
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if result.FilePath != tt.wantPath {
				t.Errorf("Resolve() FilePath = %v, want %v", result.FilePath, tt.wantPath)
			}
			if result.ModName != tt.wantName {
				t.Errorf("Resolve() ModName = %v, want %v", result.ModName, tt.wantName)
			}
		})
	}
}

func TestNormalizeRequireName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"foo", "foo"},
		{"foo.bar", "foo/bar"},
		{"foo.bar.baz", "foo/bar/baz"},
		{"./foo", "./foo"},
		{"foo/bar", "foo/bar"},
		{"../foo", "../foo"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeRequireName(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizeRequireName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestResolverWithPrefix(t *testing.T) {
	tmpDir := t.TempDir()

	// Create root directory with hyphenated name for auto-detection test
	rootDir := filepath.Join(tmpDir, "test-project")
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create lib directory inside root
	libDir := filepath.Join(rootDir, "lib")
	if err := os.MkdirAll(libDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write test Lua file: lib/tuple_config.lua
	tupleConfigPath := filepath.Join(libDir, "tuple_config.lua")
	if err := os.WriteFile(tupleConfigPath, []byte("-- tuple config"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test with explicit strip prefix
	resolver := NewWithPrefix(rootDir, []string{libDir}, "?.lua;?/init.lua", "tuple_diff")

	tests := []struct {
		name     string
		require  string
		fromDir  string
		wantPath string
		wantName string
	}{
		{
			name:     "strip explicit prefix",
			require:  "tuple_diff.lib.tuple_config",
			fromDir:  rootDir,
			wantPath: tupleConfigPath,
			wantName: "tuple_diff.lib.tuple_config",
		},
		{
			name:     "strip auto-detected prefix from root directory",
			require:  "test_project.lib.tuple_config",
			fromDir:  rootDir,
			wantPath: tupleConfigPath,
			wantName: "test_project.lib.tuple_config",
		},
		{
			name:     "strip lib prefix when searching in lib directory",
			require:  "lib.tuple_config",
			fromDir:  libDir,
			wantPath: tupleConfigPath,
			wantName: "lib.tuple_config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.Resolve(tt.require, tt.fromDir)
			if tt.wantPath == "" {
				if err == nil {
					t.Errorf("Resolve() expected error, got result %v", result)
				}
				return
			}
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if result.FilePath != tt.wantPath {
				t.Errorf("Resolve() FilePath = %v, want %v", result.FilePath, tt.wantPath)
			}
			if result.ModName != tt.wantName {
				t.Errorf("Resolve() ModName = %v, want %v", result.ModName, tt.wantName)
			}
		})
	}
}
