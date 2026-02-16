package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bigbes/lua-amalgamate/internal/config"
	"github.com/bigbes/lua-amalgamate/internal/emit"
	"github.com/bigbes/lua-amalgamate/internal/graph"
	"github.com/bigbes/lua-amalgamate/internal/parse"
	"github.com/bigbes/lua-amalgamate/internal/resolve"
	"github.com/bigbes/lua-amalgamate/internal/transform"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	koanf "github.com/knadh/koanf/v2"
	"github.com/spf13/pflag"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	var (
		configPath       string
		entry            string
		output           string
		root             string
		path             string
		search           multiString
		skip             multiString
		include          multiString
		strict           optionalBool
		removeComments   optionalBool
		removeEmptyLines optionalBool
		minify           optionalBool
		stripShebang     optionalBool
		prefix           string
		suffix           string
		packagePrefix    string
		packageName      string
		stripPrefix      string
		showVersion      bool
	)

	pflag.StringVar(&configPath, "config", "amalg.yaml", "Path to config file")
	pflag.StringVar(&entry, "entry", "", "Entry Lua file (required)")
	pflag.StringVar(&output, "output", "", "Output file, '-' for stdout (empty = use config or default)")
	pflag.StringVar(&root, "root", "", "Base directory for module resolution")
	pflag.StringVar(&path, "path", "", "Lua path templates, semicolon-separated")
	pflag.Var(&search, "search", "Additional search directory (repeatable)")
	pflag.Var(&skip, "skip", "Skip package (pattern, repeatable)")
	pflag.Var(&include, "include", "Include package (exact name, repeatable)")
	pflag.Var(&strict, "strict", "Treat unresolved requires as errors")
	pflag.Var(&removeComments, "remove-comments", "Strip Lua comments from output")
	pflag.Var(&removeEmptyLines, "remove-empty-lines", "Strip empty lines from output")
	pflag.Var(&minify, "minify", "Minify Lua source in output")
	pflag.Var(&stripShebang, "strip-shebang", "Remove shebang line (#!/...) from Lua files")
	pflag.StringVar(&prefix, "prefix", "", "Prefix Lua code inserted before modules")
	pflag.StringVar(&suffix, "suffix", "", "Suffix Lua code appended after entry require")
	pflag.StringVar(&packagePrefix, "package-prefix", "", "Prefix for all module names (e.g., 'mypkg' makes require('mypkg.module')")
	pflag.StringVar(&packageName, "package-name", "", "Package name (sets both strip-prefix and package-prefix to this value)")
	pflag.StringVar(&stripPrefix, "strip-prefix", "", "Strip prefix from module names (e.g., 'tuple_diff' makes require('tuple_diff.lib.foo') find 'lib/foo.lua')")
	pflag.BoolVar(&showVersion, "version", false, "Print version and exit")

	pflag.Parse()

	if showVersion {
		fmt.Printf("lua-amalgamate version %s (commit %s, built on %s)\n", version, commit, date)
		os.Exit(0)
	}

	// Initialize koanf with defaults, file, env, and CLI flags
	k := koanf.New(".")

	// Load defaults
	defaultCfg := config.Default()
	k.Set("entry", defaultCfg.Entry)
	k.Set("output", defaultCfg.Output)
	k.Set("root", defaultCfg.Root)
	k.Set("path", defaultCfg.Path)
	k.Set("search", defaultCfg.Search)
	k.Set("strict", defaultCfg.Strict)
	k.Set("transform.remove_comments", defaultCfg.Transform.RemoveComments)
	k.Set("transform.remove_empty_lines", defaultCfg.Transform.RemoveEmptyLines)
	k.Set("transform.minify", defaultCfg.Transform.Minify)
	k.Set("transform.strip_shebang", defaultCfg.Transform.StripShebang)
	k.Set("prefix", defaultCfg.Prefix)
	k.Set("suffix", defaultCfg.Suffix)
	k.Set("package_prefix", defaultCfg.PackagePrefix)
	k.Set("package_name", defaultCfg.PackageName)
	k.Set("strip_prefix", defaultCfg.StripPrefix)
	k.Set("skip_packages", defaultCfg.SkipPackages)
	k.Set("include_packages", defaultCfg.IncludePackages)

	// Load config file if exists
	if configPath == "" {
		configPath = "amalg.yaml"
	}

	if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "error: load config file %q: %v\n", configPath, err)
			os.Exit(1)
		}
		// File doesn't exist, continue with defaults
	}

	// Load environment variables
	if err := k.Load(env.Provider("AMALG_", ".", func(s string) string {
		s = strings.TrimPrefix(s, "AMALG_")
		s = strings.ToLower(s)
		s = strings.ReplaceAll(s, "_", ".")
		return s
	}), nil); err != nil {
		fmt.Fprintf(os.Stderr, "error: load environment: %v\n", err)
		os.Exit(1)
	}

	// Load CLI flags via posflag provider
	if err := k.Load(posflag.Provider(pflag.CommandLine, ".", k), nil); err != nil {
		fmt.Fprintf(os.Stderr, "error: load CLI flags: %v\n", err)
		os.Exit(1)
	}

	// Unmarshal config
	var cfg config.Config
	if err := k.Unmarshal("", &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: unmarshal config: %v\n", err)
		os.Exit(1)
	}

	// Apply optionalBool overrides (since koanf can't handle tri-state bools)
	if strict.set {
		cfg.Strict = strict.value
	}
	if removeComments.set {
		cfg.Transform.RemoveComments = removeComments.value
	}
	if removeEmptyLines.set {
		cfg.Transform.RemoveEmptyLines = removeEmptyLines.value
	}
	if minify.set {
		cfg.Transform.Minify = minify.value
	}
	if stripShebang.set {
		cfg.Transform.StripShebang = stripShebang.value
	}

	// Apply package_name convenience (already done in config.LoadConfig but we re-apply for CLI flags)
	if cfg.PackageName != "" {
		if cfg.StripPrefix == "" {
			cfg.StripPrefix = cfg.PackageName
		}
		if cfg.PackagePrefix == "" {
			cfg.PackagePrefix = cfg.PackageName
		}
	}

	// Entry is required
	if cfg.Entry == "" {
		fmt.Fprintln(os.Stderr, "error: --entry is required (or set in config file)")
		os.Exit(1)
	}

	if err := cfg.ResolveRoot(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	parser := parse.New()
	resolver := resolve.NewWithPrefix(cfg.Root, cfg.Search, cfg.Path, cfg.StripPrefix)
	g, err := graph.Build(&cfg, parser, resolver)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	for _, w := range g.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %s:%d: %s\n", w.File, w.Line, w.Message)
	}

	transforms := transform.BuildPipeline(cfg.Transform)

	var out io.Writer = os.Stdout
	if cfg.Output != "-" {
		dir := filepath.Dir(cfg.Output)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "error: create output directory: %v\n", err)
			os.Exit(1)
		}
		f, err := os.Create(cfg.Output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: create output file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		out = f
	}

	if err := emit.Emit(out, g, transforms, cfg.Prefix, cfg.Suffix); err != nil {
		fmt.Fprintf(os.Stderr, "error: emit: %v\n", err)
		os.Exit(1)
	}
}

type multiString []string

func (m *multiString) String() string {
	return fmt.Sprintf("%v", []string(*m))
}

func (m *multiString) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func (m *multiString) Type() string {
	return "string"
}

type optionalBool struct {
	set   bool
	value bool
}

func (b *optionalBool) Set(s string) error {
	v, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}
	b.set = true
	b.value = v
	return nil
}

func (b *optionalBool) String() string {
	if !b.set {
		return ""
	}
	return strconv.FormatBool(b.value)
}

func (b *optionalBool) Type() string {
	return "bool"
}

func (b *optionalBool) IsBoolFlag() bool {
	return true
}
