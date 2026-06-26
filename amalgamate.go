// Package amalgamate merges a multi-file Lua project into a single
// self-contained script, preserving require() semantics (run-once caching,
// circular dependencies, module identity).
//
// It is the public API of lua-amalgamate, intended for embedding into other
// tools. The command-line interface lives in cmd/lua-amalgamate and is a thin
// wrapper over this package.
//
//	var buf bytes.Buffer
//	opts := amalgamate.DefaultOptions()
//	opts.Entry = "src/main.lua"
//	res, err := amalgamate.Bundle(opts, &buf)
//	if err != nil { /* ... */ }
//	for _, w := range res.Warnings { /* report w */ }
package amalgamate

import (
	"errors"
	"io"

	"github.com/bigbes/lua-amalgamate/internal/config"
	"github.com/bigbes/lua-amalgamate/internal/emit"
	"github.com/bigbes/lua-amalgamate/internal/graph"
	"github.com/bigbes/lua-amalgamate/internal/parse"
	"github.com/bigbes/lua-amalgamate/internal/resolve"
	"github.com/bigbes/lua-amalgamate/internal/transform"
)

// ErrNoEntry is returned by Bundle when Options.Entry is empty.
var ErrNoEntry = errors.New("amalgamate: entry not set")

// ErrModuleNotFound is the cause wrapped when a required module cannot be
// located. In strict mode it is wrapped by an *UnresolvedError; check either
// with errors.Is(err, amalgamate.ErrModuleNotFound).
var ErrModuleNotFound = resolve.ErrModuleNotFound

// UnresolvedError is returned by Bundle in strict mode when a required module
// cannot be found. Recover it with errors.As, or test the cause with
// errors.Is(err, ErrModuleNotFound).
type UnresolvedError = graph.UnresolvedError

// WarningKind classifies a Warning; see the Warn* constants.
type WarningKind = graph.WarningKind

// Warning kinds, re-exported from the graph package.
const (
	WarnDynamicRequire = graph.WarnDynamicRequire
	WarnSkipped        = graph.WarnSkipped
	WarnUnresolved     = graph.WarnUnresolved
	WarnCModule        = graph.WarnCModule
	WarnNonLua         = graph.WarnNonLua
)

// Options configures a bundling run. The zero value is not valid — at minimum
// Entry must be set; prefer starting from DefaultOptions. The Output field is
// ignored by Bundle (which writes to the io.Writer it is given) and exists for
// callers that load options from a config file.
type Options = config.Config

// TransformOptions controls source rewriting applied to each module before it
// is embedded (comment/blank-line/shebang stripping and minification).
type TransformOptions = config.TransformConfig

// Warning describes a non-fatal issue found while building the dependency
// graph, such as a dynamic or unresolved require that could not be bundled.
type Warning = graph.Warning

// Result is returned by Bundle alongside the written output.
type Result struct {
	// Warnings collected while resolving the dependency graph. Non-fatal; the
	// bundle is still produced. In strict mode an unresolved require is a hard
	// error instead and Bundle returns it.
	Warnings []Warning
}

// DefaultOptions returns Options populated with the same defaults the CLI uses
// (stdout output, `?.lua;?/init.lua` path, arg-fix on, all transforms off).
func DefaultOptions() Options {
	return config.Default()
}

// LoadOptions reads Options from a YAML config file (and AMALG_* environment
// variables) layered over the defaults. A missing file is not an error — the
// defaults plus environment are returned. Pass "" to use the default path
// (amalg.yaml).
func LoadOptions(path string) (Options, error) {
	return config.LoadConfig(path)
}

// Bundle amalgamates the project described by opts and writes the resulting Lua
// source to w. It resolves the project root and the package_name convenience
// itself, so a caller only needs to set the fields it cares about. opts is
// taken by value and not mutated.
func Bundle(opts Options, w io.Writer) (*Result, error) {
	if opts.Entry == "" {
		return nil, ErrNoEntry
	}
	opts.ApplyPackageName()
	if err := opts.ResolveRoot(); err != nil {
		return nil, err
	}

	parser := parse.New()
	resolver := resolve.NewWithPrefix(opts.Root, opts.Search, opts.Path, opts.StripPrefix)
	g, err := graph.Build(&opts, parser, resolver)
	if err != nil {
		return nil, err
	}

	transforms := transform.BuildPipeline(opts.Transform)
	emitOpts := emit.Options{
		Prefix:   opts.Prefix,
		Suffix:   opts.Suffix,
		Shebang:  opts.Shebang,
		Debug:    opts.Debug,
		Fallback: opts.Fallback,
		NoArgFix: !opts.ArgFix,
	}
	if err := emit.Emit(w, g, transforms, emitOpts); err != nil {
		return nil, err
	}

	return &Result{Warnings: g.Warnings}, nil
}
