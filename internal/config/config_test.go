package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()
	assert.Equal(t, "-", cfg.Output, "Default().Output = %q, want \"-\"", cfg.Output)
	assert.Equal(t, "?.lua;?/init.lua", cfg.Path, "Default().Path = %q, want %q", cfg.Path, "?.lua;?/init.lua")
	assert.False(t, cfg.Strict, "Default().Strict = %v, want false", cfg.Strict)
	assert.Len(t, cfg.Search, 0, "len(Default().Search) = %d, want 0", len(cfg.Search))
}

func TestLoadConfigMissingFile(t *testing.T) {
	// Non-existent file should not error
	cfg, err := LoadConfig("/non/existent.yaml")
	require.NoError(t, err)
	// Should return defaults
	assert.Equal(t, "-", cfg.Output, "LoadConfig() Output = %q, want \"-\"", cfg.Output)
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
	require.NoError(t, os.WriteFile(configPath, []byte(yamlContent), 0644))

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)

	assert.Equal(t, "src/main.lua", cfg.Entry, "Entry = %q, want %q", cfg.Entry, "src/main.lua")
	assert.Equal(t, "dist/bundle.lua", cfg.Output, "Output = %q, want %q", cfg.Output, "dist/bundle.lua")
	assert.Equal(t, "src/", cfg.Root, "Root = %q, want %q", cfg.Root, "src/")
	assert.Equal(t, "?.lua", cfg.Path, "Path = %q, want %q", cfg.Path, "?.lua")
	assert.Equal(t, []string{"lib/", "vendor/"}, cfg.Search, "Search = %v, want %v", cfg.Search, []string{"lib/", "vendor/"})
	assert.True(t, cfg.Strict, "Strict = false, want true")
	assert.True(t, cfg.Transform.RemoveComments, "Transform.RemoveComments = false, want true")
	assert.False(t, cfg.Transform.RemoveEmptyLines, "Transform.RemoveEmptyLines = true, want false")
	assert.False(t, cfg.Transform.Minify, "Transform.Minify = true, want false")
	assert.True(t, cfg.Transform.StripShebang, "Transform.StripShebang = false, want true")
	assert.Equal(t, "print(\"prefix\")\n", cfg.Prefix, "Prefix = %q, want %q", cfg.Prefix, "print(\"prefix\")\\n")
	assert.Equal(t, "print(\"suffix\")\n", cfg.Suffix, "Suffix = %q, want %q", cfg.Suffix, "print(\"suffix\")\\n")
}

func TestLoadConfigWithStripPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "amalg.yaml")
	yamlContent := `entry: src/main.lua
strip_prefix: "tuple_diff"
package_prefix: "mypkg"
`
	require.NoError(t, os.WriteFile(configPath, []byte(yamlContent), 0644))

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)

	assert.Equal(t, "tuple_diff", cfg.StripPrefix, "StripPrefix = %q, want %q", cfg.StripPrefix, "tuple_diff")
	assert.Equal(t, "mypkg", cfg.PackagePrefix, "PackagePrefix = %q, want %q", cfg.PackagePrefix, "mypkg")
}

func TestResolveRoot(t *testing.T) {
	tmpDir := t.TempDir()
	entryPath := filepath.Join(tmpDir, "src", "main.lua")
	require.NoError(t, os.MkdirAll(filepath.Dir(entryPath), 0755))

	cfg := Config{
		Entry: entryPath,
		// Root not set
	}
	require.NoError(t, cfg.ResolveRoot())
	expectedRoot := filepath.Dir(entryPath)
	assert.Equal(t, expectedRoot, cfg.Root, "Root = %q, want %q", cfg.Root, expectedRoot)
}

func TestResolveRootWithEntryMissing(t *testing.T) {
	cfg := Config{
		Entry: "",
	}
	err := cfg.ResolveRoot()
	require.Error(t, err, "ResolveRoot() expected error")
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
			assert.Equal(t, tt.expected, got, "ShouldSkip(%q) = %v, want %v", tt.name, got, tt.expected)
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
	require.NoError(t, os.WriteFile(configPath, []byte(yamlContent), 0644))

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)

	assert.Equal(t, "mypkg", cfg.PackageName, "PackageName = %q, want %q", cfg.PackageName, "mypkg")
	assert.Equal(t, "mypkg", cfg.StripPrefix, "StripPrefix = %q, want %q", cfg.StripPrefix, "mypkg")
	assert.Equal(t, "mypkg", cfg.PackagePrefix, "PackagePrefix = %q, want %q", cfg.PackagePrefix, "mypkg")

	// Test 2: explicit strip_prefix overrides package_name default
	configPath2 := filepath.Join(tmpDir, "amalg2.yaml")
	yamlContent2 := `entry: src/main.lua
package_name: "mypkg"
strip_prefix: "custom"
`
	require.NoError(t, os.WriteFile(configPath2, []byte(yamlContent2), 0644))

	cfg2, err := LoadConfig(configPath2)
	require.NoError(t, err)

	assert.Equal(t, "mypkg", cfg2.PackageName, "PackageName = %q, want %q", cfg2.PackageName, "mypkg")
	assert.Equal(t, "custom", cfg2.StripPrefix, "StripPrefix = %q, want %q", cfg2.StripPrefix, "custom")
	assert.Equal(t, "mypkg", cfg2.PackagePrefix, "PackagePrefix = %q, want %q", cfg2.PackagePrefix, "mypkg")

	// Test 3: explicit package_prefix overrides package_name default
	configPath3 := filepath.Join(tmpDir, "amalg3.yaml")
	yamlContent3 := `entry: src/main.lua
package_name: "mypkg"
package_prefix: "custom"
`
	require.NoError(t, os.WriteFile(configPath3, []byte(yamlContent3), 0644))

	cfg3, err := LoadConfig(configPath3)
	require.NoError(t, err)

	assert.Equal(t, "mypkg", cfg3.PackageName, "PackageName = %q, want %q", cfg3.PackageName, "mypkg")
	assert.Equal(t, "mypkg", cfg3.StripPrefix, "StripPrefix = %q, want %q", cfg3.StripPrefix, "mypkg")
	assert.Equal(t, "custom", cfg3.PackagePrefix, "PackagePrefix = %q, want %q", cfg3.PackagePrefix, "custom")
}

func TestIncludePackages(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "amalg.yaml")
	yamlContent := `entry: src/main.lua
include_packages:
  - "plugin.optional"
  - "utils.debug"
`
	require.NoError(t, os.WriteFile(configPath, []byte(yamlContent), 0644))
	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	assert.Len(t, cfg.IncludePackages, 2, "len(IncludePackages) = %d, want 2", len(cfg.IncludePackages))
	assert.Equal(t, "plugin.optional", cfg.IncludePackages[0], "IncludePackages[0] = %q, want %q", cfg.IncludePackages[0], "plugin.optional")
	assert.Equal(t, "utils.debug", cfg.IncludePackages[1], "IncludePackages[1] = %q, want %q", cfg.IncludePackages[1], "utils.debug")
	// Ensure other fields are default
	assert.Len(t, cfg.SkipPackages, 0, "SkipPackages = %v, want empty slice", cfg.SkipPackages)
}
