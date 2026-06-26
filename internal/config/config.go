package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	koanf "github.com/knadh/koanf/v2"
)

type TransformConfig struct {
	RemoveComments   bool `yaml:"remove_comments" mapstructure:"remove_comments" koanf:"remove_comments"`
	RemoveEmptyLines bool `yaml:"remove_empty_lines" mapstructure:"remove_empty_lines" koanf:"remove_empty_lines"`
	Minify           bool `yaml:"minify" mapstructure:"minify" koanf:"minify"`
	StripShebang     bool `yaml:"strip_shebang" mapstructure:"strip_shebang" koanf:"strip_shebang"`
}

type Config struct {
	Entry           string          `yaml:"entry" mapstructure:"entry" koanf:"entry"`
	Output          string          `yaml:"output" mapstructure:"output" koanf:"output"`
	Root            string          `yaml:"root" mapstructure:"root" koanf:"root"`
	Path            string          `yaml:"path" mapstructure:"path" koanf:"path"`
	Search          []string        `yaml:"search" mapstructure:"search" koanf:"search"`
	Strict          bool            `yaml:"strict" mapstructure:"strict" koanf:"strict"`
	Debug           bool            `yaml:"debug" mapstructure:"debug" koanf:"debug"`
	Fallback        bool            `yaml:"fallback" mapstructure:"fallback" koanf:"fallback"`
	ArgFix          bool            `yaml:"arg_fix" mapstructure:"arg_fix" koanf:"arg_fix"`
	Transform       TransformConfig `yaml:"transform" mapstructure:"transform" koanf:"transform"`
	Prefix          string          `yaml:"prefix" mapstructure:"prefix" koanf:"prefix"`
	Suffix          string          `yaml:"suffix" mapstructure:"suffix" koanf:"suffix"`
	Shebang         string          `yaml:"shebang" mapstructure:"shebang" koanf:"shebang"`
	PackagePrefix   string          `yaml:"package_prefix" mapstructure:"package_prefix" koanf:"package_prefix"`
	PackageName     string          `yaml:"package_name" mapstructure:"package_name" koanf:"package_name"`
	StripPrefix     string          `yaml:"strip_prefix" mapstructure:"strip_prefix" koanf:"strip_prefix"`
	SkipPackages    []string        `yaml:"skip_packages" mapstructure:"skip_packages" koanf:"skip_packages"`
	IncludePackages []string        `yaml:"include_packages" mapstructure:"include_packages" koanf:"include_packages"`
}

func Default() Config {
	return Config{
		Output:   "-",
		Path:     "?.lua;?/init.lua",
		Search:   []string{},
		Strict:   false,
		Debug:    false,
		Fallback: false,
		ArgFix:   true,
		Transform: TransformConfig{
			RemoveComments:   false,
			RemoveEmptyLines: false,
			Minify:           false,
			StripShebang:     false,
		},
		Prefix:          "",
		Suffix:          "",
		Shebang:         "",
		PackagePrefix:   "",
		PackageName:     "",
		StripPrefix:     "",
		SkipPackages:    []string{},
		IncludePackages: []string{},
	}
}

// EnvKeyMap maps an AMALG_-prefixed environment variable name to its koanf key.
// The only nested config section is `transform`, so AMALG_TRANSFORM_X becomes
// transform.x; every other key keeps its underscores intact (e.g.
// AMALG_STRIP_PREFIX -> strip_prefix, AMALG_ARG_FIX -> arg_fix). A blanket
// "_" -> "." replacement would mangle every multi-word key.
func EnvKeyMap(s string) string {
	s = strings.ToLower(strings.TrimPrefix(s, "AMALG_"))
	if rest, ok := strings.CutPrefix(s, "transform_"); ok {
		return "transform." + rest
	}
	return s
}

func LoadConfig(configPath string) (Config, error) {
	if configPath == "" {
		configPath = "amalg.yaml"
	}

	k := koanf.New(".")

	// Set defaults
	defaultCfg := Default()
	k.Set("entry", defaultCfg.Entry)
	k.Set("output", defaultCfg.Output)
	k.Set("root", defaultCfg.Root)
	k.Set("path", defaultCfg.Path)
	k.Set("search", defaultCfg.Search)
	k.Set("strict", defaultCfg.Strict)
	k.Set("debug", defaultCfg.Debug)
	k.Set("fallback", defaultCfg.Fallback)
	k.Set("arg_fix", defaultCfg.ArgFix)
	k.Set("transform.remove_comments", defaultCfg.Transform.RemoveComments)
	k.Set("transform.remove_empty_lines", defaultCfg.Transform.RemoveEmptyLines)
	k.Set("transform.minify", defaultCfg.Transform.Minify)
	k.Set("transform.strip_shebang", defaultCfg.Transform.StripShebang)
	k.Set("prefix", defaultCfg.Prefix)
	k.Set("suffix", defaultCfg.Suffix)
	k.Set("shebang", defaultCfg.Shebang)
	k.Set("package_prefix", defaultCfg.PackagePrefix)
	k.Set("package_name", defaultCfg.PackageName)
	k.Set("strip_prefix", defaultCfg.StripPrefix)
	k.Set("skip_packages", defaultCfg.SkipPackages)
	k.Set("include_packages", defaultCfg.IncludePackages)

	// Load YAML config file if exists
	if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
		if !os.IsNotExist(err) {
			return Config{}, fmt.Errorf("load config file %q: %w", configPath, err)
		}
		// File doesn't exist, continue with defaults
	}

	// Load environment variables
	if err := k.Load(env.Provider("AMALG_", ".", EnvKeyMap), nil); err != nil {
		return Config{}, fmt.Errorf("load environment: %w", err)
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	// Apply package_name convenience
	if cfg.PackageName != "" {
		if cfg.StripPrefix == "" {
			cfg.StripPrefix = cfg.PackageName
		}
		if cfg.PackagePrefix == "" {
			cfg.PackagePrefix = cfg.PackageName
		}
	}

	return cfg, nil
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
