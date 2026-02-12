package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()
	if cfg.Output != "-" {
		t.Errorf("Default().Output = %q, want \"-\"", cfg.Output)
	}
	if cfg.Path != "?.lua;?/init.lua" {
		t.Errorf("Default().Path = %q, want %q", cfg.Path, "?.lua;?/init.lua")
	}
	if cfg.Strict != false {
		t.Errorf("Default().Strict = %v, want false", cfg.Strict)
	}
	if len(cfg.Search) != 0 {
		t.Errorf("len(Default().Search) = %d, want 0", len(cfg.Search))
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	// Non-existent file should not error
	cfg, err := LoadConfig("/non/existent.yaml")
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	// Should return defaults
	if cfg.Output != "-" {
		t.Errorf("LoadConfig() Output = %q, want \"-\"", cfg.Output)
	}
}

func TestLoadConfigValidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "amalg.yaml")
	yamlContent := `entry: src/main.lua
output: dist/bundle.lua
root: src/
path: "?.lua"
search:
  - lib/
  - vendor/
strict: true
prefix: |
  print("prefix")
suffix: |
  print("suffix")
transform:
  remove_comments: true
  remove_empty_lines: false
  minify: false
  strip_shebang: true
`
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Entry != "src/main.lua" {
		t.Errorf("Entry = %q, want %q", cfg.Entry, "src/main.lua")
	}
	if cfg.Output != "dist/bundle.lua" {
		t.Errorf("Output = %q, want %q", cfg.Output, "dist/bundle.lua")
	}
	if cfg.Root != "src/" {
		t.Errorf("Root = %q, want %q", cfg.Root, "src/")
	}
	if cfg.Path != "?.lua" {
		t.Errorf("Path = %q, want %q", cfg.Path, "?.lua")
	}
	if len(cfg.Search) != 2 || cfg.Search[0] != "lib/" || cfg.Search[1] != "vendor/" {
		t.Errorf("Search = %v, want %v", cfg.Search, []string{"lib/", "vendor/"})
	}
	if !cfg.Strict {
		t.Error("Strict = false, want true")
	}
	if !cfg.Transform.RemoveComments {
		t.Error("Transform.RemoveComments = false, want true")
	}
	if cfg.Transform.RemoveEmptyLines {
		t.Error("Transform.RemoveEmptyLines = true, want false")
	}
	if cfg.Transform.Minify {
		t.Error("Transform.Minify = true, want false")
	}
	if !cfg.Transform.StripShebang {
		t.Error("Transform.StripShebang = false, want true")
	}
	if cfg.Prefix != "print(\"prefix\")\n" {
		t.Errorf("Prefix = %q, want %q", cfg.Prefix, "print(\"prefix\")\\n")
	}
	if cfg.Suffix != "print(\"suffix\")\n" {
		t.Errorf("Suffix = %q, want %q", cfg.Suffix, "print(\"suffix\")\\n")
	}
}

func TestLoadConfigWithStripPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "amalg.yaml")
	yamlContent := `entry: src/main.lua
strip_prefix: "tuple_diff"
package_prefix: "mypkg"
`
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.StripPrefix != "tuple_diff" {
		t.Errorf("StripPrefix = %q, want %q", cfg.StripPrefix, "tuple_diff")
	}
	if cfg.PackagePrefix != "mypkg" {
		t.Errorf("PackagePrefix = %q, want %q", cfg.PackagePrefix, "mypkg")
	}
}

func TestResolveRoot(t *testing.T) {
	tmpDir := t.TempDir()
	entryPath := filepath.Join(tmpDir, "src", "main.lua")
	if err := os.MkdirAll(filepath.Dir(entryPath), 0755); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		Entry: entryPath,
		// Root not set
	}
	if err := cfg.ResolveRoot(); err != nil {
		t.Fatalf("ResolveRoot() error = %v", err)
	}
	expectedRoot := filepath.Dir(entryPath)
	if cfg.Root != expectedRoot {
		t.Errorf("Root = %q, want %q", cfg.Root, expectedRoot)
	}
}

func TestResolveRootWithEntryMissing(t *testing.T) {
	cfg := Config{
		Entry: "",
	}
	err := cfg.ResolveRoot()
	if err == nil {
		t.Fatal("ResolveRoot() expected error")
	}
}

func TestShouldSkip(t *testing.T) {
	cfg := Config{
		SkipPackages: []string{"xlog.*", "yaml", "cjson"},
	}

	tests := []struct {
		name     string
		expected bool
	}{
		{"xlog.core", true},
		{"xlog.util", true},
		{"xlog", true}, // prefix match without dot also matches
		{"xlogextra", false},
		{"yaml", true},
		{"yaml.util", false},
		{"cjson", true},
		{"json", false},
		{"other", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.ShouldSkip(tt.name)
			if got != tt.expected {
				t.Errorf("ShouldSkip(%q) = %v, want %v", tt.name, got, tt.expected)
			}
		})
	}
}

func TestPackageNameConvenience(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "amalg.yaml")

	// Test 1: package_name sets both strip_prefix and package_prefix when not specified
	yamlContent := `entry: src/main.lua
package_name: "mypkg"
`
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.PackageName != "mypkg" {
		t.Errorf("PackageName = %q, want %q", cfg.PackageName, "mypkg")
	}
	if cfg.StripPrefix != "mypkg" {
		t.Errorf("StripPrefix = %q, want %q", cfg.StripPrefix, "mypkg")
	}
	if cfg.PackagePrefix != "mypkg" {
		t.Errorf("PackagePrefix = %q, want %q", cfg.PackagePrefix, "mypkg")
	}

	// Test 2: explicit strip_prefix overrides package_name default
	configPath2 := filepath.Join(tmpDir, "amalg2.yaml")
	yamlContent2 := `entry: src/main.lua
package_name: "mypkg"
strip_prefix: "custom"
`
	if err := os.WriteFile(configPath2, []byte(yamlContent2), 0644); err != nil {
		t.Fatal(err)
	}

	cfg2, err := LoadConfig(configPath2)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg2.PackageName != "mypkg" {
		t.Errorf("PackageName = %q, want %q", cfg2.PackageName, "mypkg")
	}
	if cfg2.StripPrefix != "custom" {
		t.Errorf("StripPrefix = %q, want %q", cfg2.StripPrefix, "custom")
	}
	if cfg2.PackagePrefix != "mypkg" {
		t.Errorf("PackagePrefix = %q, want %q", cfg2.PackagePrefix, "mypkg")
	}

	// Test 3: explicit package_prefix overrides package_name default
	configPath3 := filepath.Join(tmpDir, "amalg3.yaml")
	yamlContent3 := `entry: src/main.lua
package_name: "mypkg"
package_prefix: "custom"
`
	if err := os.WriteFile(configPath3, []byte(yamlContent3), 0644); err != nil {
		t.Fatal(err)
	}

	cfg3, err := LoadConfig(configPath3)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg3.PackageName != "mypkg" {
		t.Errorf("PackageName = %q, want %q", cfg3.PackageName, "mypkg")
	}
	if cfg3.StripPrefix != "mypkg" {
		t.Errorf("StripPrefix = %q, want %q", cfg3.StripPrefix, "mypkg")
	}
	if cfg3.PackagePrefix != "custom" {
		t.Errorf("PackagePrefix = %q, want %q", cfg3.PackagePrefix, "custom")
	}
}

func TestIncludePackages(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "amalg.yaml")
	yamlContent := `entry: src/main.lua
include_packages:
  - "plugin.optional"
  - "utils.debug"
`
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if len(cfg.IncludePackages) != 2 {
		t.Errorf("len(IncludePackages) = %d, want 2", len(cfg.IncludePackages))
	}
	if cfg.IncludePackages[0] != "plugin.optional" {
		t.Errorf("IncludePackages[0] = %q, want %q", cfg.IncludePackages[0], "plugin.optional")
	}
	if cfg.IncludePackages[1] != "utils.debug" {
		t.Errorf("IncludePackages[1] = %q, want %q", cfg.IncludePackages[1], "utils.debug")
	}
	// Ensure other fields are default
	if len(cfg.SkipPackages) != 0 {
		t.Errorf("SkipPackages = %v, want empty slice", cfg.SkipPackages)
	}
}
