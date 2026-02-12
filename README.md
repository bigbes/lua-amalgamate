# Amalgamator – Lua Bundler

Amalgamator merges multi‑file Lua projects into a single self‑contained `.lua` file while preserving exact `require()` semantics (run‑once caching, circular dependencies, etc.).

## How It Works

Instead of inlining module source at call sites, amalgamator registers each module into Lua's native `package.preload` table. This ensures:

- **Standard `require()` works unchanged** – no custom override needed
- **Modules execute lazily** – only when first required
- **Caching handled by Lua** – `package.loaded` works as usual
- **Circular dependencies safe** – Lua's own mechanism breaks cycles
- **Each loader receives `...` (the module name)** matching standard behavior

## Installation

```bash
go install github.com/bigbes/lua-amalgamator/cmd/amalg@latest
```

Or build from source:

```bash
git clone https://github.com/bigbes/lua-amalgamator
cd amalgamator
go install ./cmd/amalg
```

## Quick Start

Given a Lua project:

```
project/
├── main.lua
└── lib/
    └── utils.lua
```

```bash
amalg --entry project/main.lua --output bundle.lua
```

The resulting `bundle.lua` contains both modules wrapped in `package.preload` assignments and a final `require("main")` to start execution.

## Configuration

Amalgamator can be configured via YAML file (`amalg.yaml`) and/or CLI flags (flags override YAML).

Example `amalg.yaml`:

```yaml
entry: src/main.lua
output: dist/bundle.lua
root: src/
path: "?.lua;?/init.lua"
search:
  - lib/
  - vendor/
strict: false
transform:
  remove_comments: true
  remove_empty_lines: true
  minify: false
```

All CLI flags:

| Flag | Description | Default |
|------|-------------|---------|
| `--config` | Path to config file | `amalg.yaml` |
| `--entry` | Entry Lua file **(required)** | – |
| `--output` | Output file, `-` for stdout | `-` |
| `--root` | Base directory for module resolution | directory of entry file |
| `--path` | Lua path templates, semicolon‑separated | `?.lua;?/init.lua` |
| `--search` | Additional search directory (repeatable) | – |
| `--skip` | Skip package (pattern, repeatable) | – |
| `--include` | Include package (exact name, repeatable) | – |
| `--strict` | Treat unresolved requires as errors | `false` |
| `--remove‑comments` | Strip Lua comments from output | `false` |
| `--remove‑empty‑lines` | Strip empty lines from output | `false` |
| `--minify` | Minify Lua source (implies comment/empty‑line removal) | `false` |
| `--strip‑shebang` | Remove shebang line (`#!/...`) from Lua files | `false` |
| `--prefix` | Prefix Lua code inserted before modules | – |
| `--suffix` | Suffix Lua code appended after entry require | – |
| `--package‑prefix` | Prefix for all module names (e.g., `mypkg` enables `require("mypkg.module")`) | – |
| `--package‑name` | Package name (sets both `--strip‑prefix` and `--package‑prefix` to this value) | – |
| `--strip‑prefix` | Strip prefix from module names (e.g., `tuple_diff` makes `require("tuple_diff.lib.foo")` find `lib/foo.lua`) | – |

## Module Resolution

Amalgamator resolves `require("foo.bar")` exactly like Lua's `package.searchpath`:

1. Dots are replaced by the OS path separator (`foo/bar`).
2. Each search location (entry directory, root, additional `search` directories) is tried.
3. For each location, each template from `path` is applied (e.g., `?.lua`, `?/init.lua`).
4. The first matching file is used.

Path‑style requires (`require("./foo/bar")`) and relative requires are supported.

### Prefix Stripping

Amalgamator can automatically strip prefixes from module names, which is useful for projects where module names include a package prefix that doesn't match directory structure:

- **Explicit prefix**: use `--strip‑prefix tuple_diff` to strip `tuple_diff.` from module names before resolution.
- **Auto‑detection**: if the root directory name contains hyphens (e.g., `tuple‑diff`), it's converted to underscores (`tuple_diff`) and used as a prefix automatically.
- **Search‑directory prefix**: when searching in a directory like `lib/`, the prefix `lib.` is also stripped from module names (e.g., `lib.tuple_config` → `tuple_config`).

Prefix stripping helps resolve modules like `require("tuple_diff.lib.tuple_config")` to file `lib/tuple_config.lua` without requiring a `tuple_diff/` subdirectory.

For library projects where modules are named with a package prefix (e.g., `mypkg.utils`) but files are stored without that prefix in the filesystem, use `--package‑name mypkg`. This both strips the prefix during resolution and adds it back to output names, making the bundled library usable as `require("mypkg.utils")`.

## Output Format

```lua
-- Amalgamated by amalg
-- Entry: main

package.preload["foo.bar"] = function(...)
  -- original module source
end

package.preload["main"] = function(...)
  -- entry module source
end

require("main")
```

## Limitations (v1)

- **Dynamic requires** (`require(variable)`) are detected but produce a warning; the module is not bundled.
- **C modules** (`.so`/`.dll`) produce a warning and are left as runtime requires.
- **Lua 5.1 syntax only** (gopher‑lua parser). Lua 5.2+ features may cause parse errors.
- **No source maps** or debug info preservation.
- **No asset embedding** (only `.lua` files are bundled).

## Development

See [AGENTS.md](AGENTS.md) for build commands, code style, and contribution guidelines.

## License

MIT