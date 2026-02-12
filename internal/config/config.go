package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type TransformConfig struct {
	RemoveComments   bool `yaml:"remove_comments"`
	RemoveEmptyLines bool `yaml:"remove_empty_lines"`
	Minify           bool `yaml:"minify"`
	StripShebang     bool `yaml:"strip_shebang"`
}

type Config struct {
	Entry           string          `yaml:"entry"`
	Output          string          `yaml:"output"`
	Root            string          `yaml:"root"`
	Path            string          `yaml:"path"`
	Search          []string        `yaml:"search"`
	Strict          bool            `yaml:"strict"`
	Transform       TransformConfig `yaml:"transform"`
	Prefix          string          `yaml:"prefix"`
	Suffix          string          `yaml:"suffix"`
	PackagePrefix   string          `yaml:"package_prefix"`
	PackageName     string          `yaml:"package_name"`
	StripPrefix     string          `yaml:"strip_prefix"`
	SkipPackages    []string        `yaml:"skip_packages"`
	IncludePackages []string        `yaml:"include_packages"`
}

func Default() Config {
	return Config{
		Output: "-",
		Path:   "?.lua;?/init.lua",
		Search: []string{},
		Strict: false,
		Transform: TransformConfig{
			RemoveComments:   false,
			RemoveEmptyLines: false,
			Minify:           false,
			StripShebang:     false,
		},
		Prefix:          "",
		Suffix:          "",
		PackagePrefix:   "",
		PackageName:     "",
		StripPrefix:     "",
		SkipPackages:    []string{},
		IncludePackages: []string{},
	}
}

func LoadConfig(configPath string) (Config, error) {
	cfg := Default()

	if configPath == "" {
		configPath = "amalg.yaml"
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return Config{}, fmt.Errorf("read config file %q: %w", configPath, err)
	}

	var yamlCfg Config
	if err := yaml.Unmarshal(data, &yamlCfg); err != nil {
		return Config{}, fmt.Errorf("parse config file %q: %w", configPath, err)
	}

	mergeConfig(&cfg, yamlCfg)
	return cfg, nil
}

func mergeConfig(dst *Config, src Config) {
	if src.Entry != "" {
		dst.Entry = src.Entry
	}
	if src.Output != "" {
		dst.Output = src.Output
	}
	if src.Root != "" {
		dst.Root = src.Root
	}
	if src.Path != "" {
		dst.Path = src.Path
	}
	if len(src.Search) > 0 {
		dst.Search = src.Search
	}
	if src.Strict {
		dst.Strict = src.Strict
	}
	if src.Prefix != "" {
		dst.Prefix = src.Prefix
	}
	if src.Suffix != "" {
		dst.Suffix = src.Suffix
	}
	if src.PackagePrefix != "" {
		dst.PackagePrefix = src.PackagePrefix
	}
	if src.PackageName != "" {
		dst.PackageName = src.PackageName
	}
	if src.StripPrefix != "" {
		dst.StripPrefix = src.StripPrefix
	}
	if len(src.SkipPackages) > 0 {
		dst.SkipPackages = src.SkipPackages
	}
	if len(src.IncludePackages) > 0 {
		dst.IncludePackages = src.IncludePackages
	}
	if src.Transform.RemoveComments {
		dst.Transform.RemoveComments = src.Transform.RemoveComments
	}
	if src.Transform.RemoveEmptyLines {
		dst.Transform.RemoveEmptyLines = src.Transform.RemoveEmptyLines
	}
	if src.Transform.Minify {
		dst.Transform.Minify = src.Transform.Minify
	}
	if src.Transform.StripShebang {
		dst.Transform.StripShebang = src.Transform.StripShebang
	}

	// Apply package_name convenience: if set and strip_prefix/package_prefix not explicitly set, use it
	if dst.PackageName != "" {
		if dst.StripPrefix == "" {
			dst.StripPrefix = dst.PackageName
		}
		if dst.PackagePrefix == "" {
			dst.PackagePrefix = dst.PackageName
		}
	}
}

func (c *Config) ResolveRoot() error {
	if c.Root != "" {
		return nil
	}
	if c.Entry == "" {
		return fmt.Errorf("cannot determine root: entry not set")
	}
	absEntry, err := filepath.Abs(c.Entry)
	if err != nil {
		return fmt.Errorf("resolve entry path: %w", err)
	}
	c.Root = filepath.Dir(absEntry)
	return nil
}

func (c *Config) ShouldSkip(name string) bool {
	for _, pattern := range c.SkipPackages {
		if pattern == name {
			return true
		}
		if strings.HasSuffix(pattern, ".*") {
			prefix := pattern[:len(pattern)-2]
			if strings.HasPrefix(name, prefix+".") || name == prefix {
				return true
			}
		}
	}
	return false
}
