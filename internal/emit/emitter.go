package emit

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/bigbes/lua-amalgamate/internal/graph"
	"github.com/bigbes/lua-amalgamate/internal/transform"
)

// Options controls how the bundle is rendered.
type Options struct {
	// Prefix is Lua code inserted before the modules.
	Prefix string
	// Suffix is Lua code appended after the entry require.
	Suffix string
	// Debug emits each module via load(source, "@path") so runtime errors
	// report the original file name and line numbers instead of offsets into
	// the bundle. The source is embedded verbatim (no reindentation).
	Debug bool
	// Fallback registers modules in package.postload behind an appended
	// searcher instead of package.preload, so an on-disk copy on package.path
	// takes precedence and the embedded module is only a fallback.
	Fallback bool
	// Shebang, when non-empty, is written as the first line of the bundle
	// (e.g. "#!/usr/bin/env lua") so the output is directly executable.
	Shebang string
	// NoArgFix omits the `local arg = _G.arg` alias from each module. The alias
	// is emitted by default to counter the implicit `arg` local that Lua 5.1
	// (LUA_COMPAT_VARARG) injects into vararg functions; set this to drop it
	// when targeting an interpreter that doesn't need it (amalg's -a).
	NoArgFix bool
}

func Emit(w io.Writer, g *graph.Graph, transforms []transform.Transformer, opts Options) error {
	prefix, suffix := opts.Prefix, opts.Suffix
	// Ensure prefix ends with newline if present
	if prefix != "" && !strings.HasSuffix(prefix, "\n") {
		prefix = prefix + "\n"
	}
	// Ensure suffix ends with newline if present
	if suffix != "" && !strings.HasSuffix(suffix, "\n") {
		suffix = suffix + "\n"
	}

	// Sort modules by primary name. Copy the slice header first so we don't
	// reorder the caller's graph as a side effect.
	modules := make([]*graph.Module, len(g.Modules))
	copy(modules, g.Modules)
	sort.Slice(modules, func(i, j int) bool {
		nameI := primaryName(modules[i])
		nameJ := primaryName(modules[j])
		return nameI < nameJ
	})

	// Build module data for template
	moduleDataList := make([]moduleData, 0, len(modules))
	for _, mod := range modules {
		source := mod.Source
		for _, t := range transforms {
			transformed, err := t.Transform(source)
			if err != nil {
				return fmt.Errorf("transform %s: %w", mod.FilePath, err)
			}
			source = transformed
		}

		var aliases []string
		if len(mod.Names) > 1 {
			aliases = mod.Names[1:]
		}
		data := moduleData{
			PrimaryName: primaryName(mod),
			AliasNames:  aliases,
			Source:      string(source),
			Path:        mod.FilePath,
		}
		moduleDataList = append(moduleDataList, data)
	}

	regTable := "preload"
	if opts.Fallback {
		regTable = "postload"
	}
	entryName := primaryName(g.Entry)
	data := templateData{
		EntryName: entryName,
		Modules:   moduleDataList,
		Prefix:    prefix,
		Suffix:    suffix,
		Shebang:   strings.TrimRight(opts.Shebang, "\n"),
		Debug:     opts.Debug,
		Fallback:  opts.Fallback,
		NoArgFix:  opts.NoArgFix,
		RegTable:  regTable,
	}
	return outputTemplate.Execute(w, data)
}

func primaryName(mod *graph.Module) string {
	if len(mod.Names) == 0 {
		return ""
	}
	return mod.Names[0]
}
