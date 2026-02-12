package emit

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/bigbes/lua-amalgamator/internal/graph"
	"github.com/bigbes/lua-amalgamator/internal/transform"
)

func Emit(w io.Writer, g *graph.Graph, transforms []transform.Transformer, prefix, suffix string) error {
	// Ensure prefix ends with newline if present
	if prefix != "" && !strings.HasSuffix(prefix, "\n") {
		prefix = prefix + "\n"
	}
	// Ensure suffix ends with newline if present
	if suffix != "" && !strings.HasSuffix(suffix, "\n") {
		suffix = suffix + "\n"
	}

	// Sort modules by primary name
	modules := g.Modules
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

		data := moduleData{
			Names:       mod.Names,
			Source:      string(source),
			IsSingle:    len(mod.Names) == 1,
			PrimaryName: primaryName(mod),
		}
		moduleDataList = append(moduleDataList, data)
	}

	entryName := primaryName(g.Entry)
	data := templateData{
		EntryName: entryName,
		Modules:   moduleDataList,
		Prefix:    prefix,
		Suffix:    suffix,
	}
	return outputTemplate.Execute(w, data)
}

func primaryName(mod *graph.Module) string {
	if len(mod.Names) == 0 {
		return ""
	}
	return mod.Names[0]
}
