package graph

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bigbes/lua-amalgamate/internal/config"
	"github.com/bigbes/lua-amalgamate/internal/parse"
	"github.com/bigbes/lua-amalgamate/internal/resolve"
	"github.com/bigbes/lua-amalgamate/internal/transform"
)

type Warning struct {
	File    string
	Line    int
	Message string
}

type Module struct {
	ID       int
	FilePath string              // canonical absolute path
	Names    []string            // all require-strings that map to this file
	Source   []byte              // raw file contents
	Requires []parse.RequireInfo // parsed require edges
}

type Graph struct {
	Modules  []*Module          // all discovered modules
	ByPath   map[string]*Module // keyed by canonical filepath
	Entry    *Module
	Warnings []Warning
}

func moduleNameFromPath(root, filePath string) string {
	if root == "" {
		// Use filename as fallback
		base := filepath.Base(filePath)
		base = strings.TrimSuffix(base, ".lua")
		base = strings.TrimSuffix(base, ".init")
		return base
	}
	rel, err := filepath.Rel(root, filePath)
	if err != nil {
		// If relative path cannot be computed, use filename
		base := filepath.Base(filePath)
		base = strings.TrimSuffix(base, ".lua")
		base = strings.TrimSuffix(base, ".init")
		return base
	}
	rel = strings.TrimSuffix(rel, ".lua")
	rel = strings.TrimSuffix(rel, ".init")
	rel = strings.ReplaceAll(rel, string(filepath.Separator), ".")
	if rel == "." {
		return "main"
	}
	return rel
}

func applyPackagePrefix(g *Graph, packagePrefix string) {
	if packagePrefix == "" {
		return
	}
	for _, mod := range g.Modules {
		// Create prefixed versions of all names, avoiding duplicates
		prefixedNames := make([]string, 0, len(mod.Names)*2)
		// Keep original names first for internal requires
		prefixedNames = append(prefixedNames, mod.Names...)
		// Track existing names for deduplication
		existing := make(map[string]bool, len(mod.Names))
		for _, name := range mod.Names {
			existing[name] = true
		}
		for _, name := range mod.Names {
			// Don't add prefix if name already starts with it
			if strings.HasPrefix(name, packagePrefix+".") {
				continue
			}
			prefixed := packagePrefix + "." + name
			if !existing[prefixed] {
				prefixedNames = append(prefixedNames, prefixed)
				existing[prefixed] = true
			}
		}
		mod.Names = prefixedNames
	}
	// For entry module, ensure a prefixed name is first so final require uses it
	if g.Entry != nil && len(g.Entry.Names) > 0 {
		// Find a prefixed name
		for i, name := range g.Entry.Names {
			if strings.HasPrefix(name, packagePrefix+".") {
				// Move this prefixed name to the front
				if i != 0 {
					g.Entry.Names = append([]string{name}, append(g.Entry.Names[:i], g.Entry.Names[i+1:]...)...)
				}
				break
			}
		}
	}
}

func Build(cfg *config.Config, parser parse.Parser, resolver *resolve.Resolver) (*Graph, error) {
	if err := cfg.ResolveRoot(); err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}

	entryPath, err := filepath.Abs(cfg.Entry)
	if err != nil {
		return nil, fmt.Errorf("resolve entry path: %w", err)
	}

	entrySource, err := os.ReadFile(entryPath)
	if err != nil {
		return nil, fmt.Errorf("read entry file: %w", err)
	}
	entrySource = transform.StripShebang(entrySource)

	g := &Graph{
		ByPath: make(map[string]*Module),
	}

	entryMod := &Module{
		ID:       0,
		FilePath: entryPath,
		Names:    []string{moduleNameFromPath(cfg.Root, entryPath)},
		Source:   entrySource,
	}
	g.Modules = append(g.Modules, entryMod)
	g.ByPath[entryPath] = entryMod
	g.Entry = entryMod

	queue := []*Module{entryMod}
	// Process included packages
	for _, incName := range cfg.IncludePackages {
		if cfg.ShouldSkip(incName) {
			g.Warnings = append(g.Warnings, Warning{
				File:    "",
				Line:    0,
				Message: fmt.Sprintf("include package %q skipped (matches skip pattern)", incName),
			})
			continue
		}
		result, err := resolver.Resolve(incName, cfg.Root)
		if err != nil {
			if cfg.Strict {
				return nil, fmt.Errorf("include package %q not found: %w", incName, err)
			}
			g.Warnings = append(g.Warnings, Warning{
				File:    "",
				Line:    0,
				Message: fmt.Sprintf("include package %q not found: %v", incName, err),
			})
			continue
		}
		ext := strings.ToLower(filepath.Ext(result.FilePath))
		if ext == ".so" || ext == ".dll" {
			g.Warnings = append(g.Warnings, Warning{
				File:    "",
				Line:    0,
				Message: fmt.Sprintf("C module %q cannot be amalgamated", incName),
			})
			continue
		}
		if ext != ".lua" {
			g.Warnings = append(g.Warnings, Warning{
				File:    "",
				Line:    0,
				Message: fmt.Sprintf("non-Lua module %q (extension %s) cannot be amalgamated", incName, ext),
			})
			continue
		}
		if existing, ok := g.ByPath[result.FilePath]; ok {
			// Deduplicate names
			found := false
			for _, n := range existing.Names {
				if n == incName {
					found = true
					break
				}
			}
			if !found {
				existing.Names = append(existing.Names, incName)
			}
			continue
		}
		source, err := os.ReadFile(result.FilePath)
		if err != nil {
			return nil, fmt.Errorf("read include module %s: %w", result.FilePath, err)
		}
		source = transform.StripShebang(source)
		newMod := &Module{
			ID:       len(g.Modules),
			FilePath: result.FilePath,
			Names:    []string{incName},
			Source:   source,
		}
		g.Modules = append(g.Modules, newMod)
		g.ByPath[result.FilePath] = newMod
		queue = append(queue, newMod)
	}
	for len(queue) > 0 {
		mod := queue[0]
		queue = queue[1:]

		requires, err := parser.Parse(mod.Source, mod.FilePath)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", mod.FilePath, err)
		}
		mod.Requires = requires

		for _, req := range requires {
			if !req.Static {
				g.Warnings = append(g.Warnings, Warning{
					File:    mod.FilePath,
					Line:    req.Line,
					Message: fmt.Sprintf("dynamic require at line %d, cannot resolve", req.Line),
				})
				continue
			}

			if cfg.ShouldSkip(req.Name) {
				g.Warnings = append(g.Warnings, Warning{
					File:    mod.FilePath,
					Line:    req.Line,
					Message: fmt.Sprintf("skipped package %q at line %d", req.Name, req.Line),
				})
				continue
			}

			result, err := resolver.Resolve(req.Name, filepath.Dir(mod.FilePath))
			if err != nil {
				if cfg.Strict {
					return nil, fmt.Errorf("unresolved require %q at %s:%d: %w",
						req.Name, mod.FilePath, req.Line, err)
				}
				g.Warnings = append(g.Warnings, Warning{
					File:    mod.FilePath,
					Line:    req.Line,
					Message: fmt.Sprintf("unresolved require %q at line %d: %v", req.Name, req.Line, err),
				})
				continue
			}

			ext := strings.ToLower(filepath.Ext(result.FilePath))
			if ext == ".so" || ext == ".dll" {
				g.Warnings = append(g.Warnings, Warning{
					File:    mod.FilePath,
					Line:    req.Line,
					Message: fmt.Sprintf("C module %q cannot be amalgamated", req.Name),
				})
				continue
			}

			if ext != ".lua" {
				g.Warnings = append(g.Warnings, Warning{
					File:    mod.FilePath,
					Line:    req.Line,
					Message: fmt.Sprintf("non-Lua module %q (extension %s) cannot be amalgamated", req.Name, ext),
				})
				continue
			}

			if existing, ok := g.ByPath[result.FilePath]; ok {
				// Deduplicate names
				found := false
				for _, n := range existing.Names {
					if n == req.Name {
						found = true
						break
					}
				}
				if !found {
					existing.Names = append(existing.Names, req.Name)
				}
				continue
			}

			source, err := os.ReadFile(result.FilePath)
			if err != nil {
				return nil, fmt.Errorf("read module %s: %w", result.FilePath, err)
			}
			source = transform.StripShebang(source)

			newMod := &Module{
				ID:       len(g.Modules),
				FilePath: result.FilePath,
				Names:    []string{req.Name},
				Source:   source,
			}
			g.Modules = append(g.Modules, newMod)
			g.ByPath[result.FilePath] = newMod
			queue = append(queue, newMod)
		}
	}

	applyPackagePrefix(g, cfg.PackagePrefix)

	return g, nil
}
