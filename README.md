# lua-amalgamate

Merge a multi-file Lua project into a single self-contained `.lua` file, preserving exact `require()` semantics — run-once caching, circular dependencies, and module identity all behave the same as the un-bundled project.

It is a Go reimplementation of the ideas in [siffiejoe/lua-amalg](https://github.com/siffiejoe/lua-amalg), with static `require()` analysis (no need to *run* your program to discover modules) and first-class transforms, debug tracebacks, and dev-override modes.

## Features

- **Static dependency analysis** — finds `require()` calls by parsing source, no runtime instrumentation.
- **Faithful `require()` semantics** — modules registered in `package.preload`, so caching, run-once, and circular deps work via Lua's own machinery.
- **Accurate tracebacks** (`--debug`) — runtime errors report the original `file:line`, not an offset into the bundle.
- **Dev overrides** (`--fallback`) — ship a bundle but let an on-disk module take precedence during development.
- **Directly executable output** (`--shebang`) — prepend a shebang and run the bundle as a program.
- **Source transforms** — strip comments, blank lines, shebangs, or minify.
- **Module-name remapping** — strip/add package prefixes so library modules resolve to a flat file layout.
- **Cross-version output** — works on Lua 5.1–5.4 and LuaJIT (the `_ENV`/`arg` handling is version-aware).

## Installation

```bash
# Latest release
go install github.com/bigbes/lua-amalgamate/cmd/lua-amalgamate@latest

# Docker
docker pull ghcr.io/bigbes/lua-amalgamate:latest

# From source
git clone https://github.com/bigbes/lua-amalgamate
cd lua-amalgamate
go install ./cmd/lua-amalgamate
```

## Quick start

Given:

```
project/
├── main.lua          -- require("lib.utils")
└── lib/
    └── utils.lua
```

```bash
lua-amalgamate --entry project/main.lua --output bundle.lua
lua bundle.lua
```

`bundle.lua` contains every reachable module wrapped in a `package.preload` loader, followed by `return require("main")` to start execution.

## How it works

Each module is registered into Lua's native `package.preload` table instead of being inlined at its call site:

```lua
-- Amalgamated by lua-amalgamate
-- Entry: main

do
local _ENV = _ENV
package.preload["lib.utils"] = function(...)
  local name = ...
  package.loaded[name] = true
  local arg = _G.arg
-- ... original module source, verbatim ...
end
end
do
local _ENV = _ENV
package.preload["main"] = function(...)
  local name = ...
  package.loaded[name] = true
  local arg = _G.arg
-- ... entry module source, verbatim ...
end
end
return require("main")
```

Why each piece is there:

- **`package.preload[name]`** — standard `require()` finds the loader here before searching the filesystem, so the embedded copy is used and caching in `package.loaded` works unchanged.
- **`local _ENV = _ENV` in the enclosing `do` block** — makes `_ENV` the loader's *first upvalue*, which Lua 5.2's `module`/`_ENV` handling depends on. It lives outside the function on purpose.
- **`package.loaded[name] = true` before the body runs** — breaks circular `require()` chains the same way Lua's own loader does.
- **`local arg = _G.arg`** — restores the global `arg` table inside vararg functions (Lua 5.1 `LUA_COMPAT_VARARG` would otherwise shadow it). Disable with `--no-arg-fix`.
- **Module source is emitted verbatim** (no reindentation) so multi-line `[[ ... ]]` string literals keep their exact contents.

A module reachable under several names (aliases) gets one real loader on its primary name; the other names delegate via `return require("primary")`, so the body executes exactly once no matter which name is required.

## CLI flags

| Flag | Description | Default |
|------|-------------|---------|
| `--config` | Path to config file | `amalg.yaml` |
| `--entry` | Entry Lua file **(required)** | – |
| `--output` | Output file, `-` for stdout | `-` |
| `--root` | Base directory for module resolution | directory of entry file |
| `--path` | Lua path templates, semicolon-separated | `?.lua;?/init.lua` |
| `--search` | Additional search directory (repeatable) | – |
| `--skip` | Exclude a package (exact name, or `name.*` for prefix; repeatable) | – |
| `--include` | Force-include a module not reached from the entry (exact name, repeatable) | – |
| `--strict` | Treat unresolved static requires as errors | `false` |
| `--debug` | Load modules via `load(src, "@file")` so tracebacks keep the original `file:line` | `false` |
| `--fallback` | Register modules in `package.postload` behind a searcher so on-disk modules win | `false` |
| `--shebang` | Shebang written as the first line of the bundle (e.g. `#!/usr/bin/env lua`) | – |
| `--no-arg-fix` | Omit the `local arg = _G.arg` alias (only needed on Lua 5.1 with `LUA_COMPAT_VARARG`) | – |
| `--remove-comments` | Strip Lua comments from output | `false` |
| `--remove-empty-lines` | Strip empty/whitespace-only lines | `false` |
| `--minify` | Minify source (subsumes comment / empty-line / shebang removal) | `false` |
| `--strip-shebang` | Remove shebang lines from input module sources | `false` |
| `--prefix` | Lua code inserted before the modules | – |
| `--suffix` | Lua code appended after the entry `require` | – |
| `--package-prefix` | Add a prefix to all module names in output (e.g. `mypkg` → `require("mypkg.module")`) | – |
| `--strip-prefix` | Strip a prefix from module names during resolution (e.g. `tuple_diff` → file `lib/foo.lua`) | – |
| `--package-name` | Convenience: sets both `--strip-prefix` and `--package-prefix` to this value | – |
| `--version` | Print version and exit | – |

Boolean flags accept a bare form (`--debug`) or an explicit value (`--debug=false`).

## Configuration

Settings are resolved with this precedence (later wins): **defaults → `amalg.yaml` → `AMALG_*` environment variables → CLI flags**.

### YAML

```yaml
entry: src/main.lua
output: dist/bundle.lua
root: src/
path: "?.lua;?/init.lua"
search:
  - lib/
  - vendor/

strict: false
debug: false        # load() each module so tracebacks keep original file:line
fallback: false     # prefer on-disk modules over embedded copies
arg_fix: true       # emit `local arg = _G.arg` (set false for amalg's -a)
shebang: ""         # e.g. "#!/usr/bin/env lua"

# prefix: |          # Lua code inserted before modules
#   print("starting")
# suffix: |          # Lua code appended after the entry require
#   print("done")

# package_name: "mypkg"     # convenience: strip + re-add the prefix
# strip_prefix: "tuple_diff"
# package_prefix: "mypkg"

# skip_packages:            # exclude from the bundle (left as runtime requires)
#   - "cjson"
#   - "xlog.*"
# include_packages:         # bundle even if not reached from entry
#   - "plugin.optional"

transform:
  remove_comments: false
  remove_empty_lines: false
  minify: false
  strip_shebang: false
```

### Environment variables

Any option can be set with the `AMALG_` prefix and the option's name upper-cased (handy in CI):

```bash
AMALG_ENTRY=src/main.lua AMALG_OUTPUT=dist/bundle.lua AMALG_STRICT=true lua-amalgamate
```

Multi-word and nested options map as you'd expect — `AMALG_STRIP_PREFIX` → `strip_prefix`, `AMALG_ARG_FIX` → `arg_fix`, and `AMALG_TRANSFORM_REMOVE_COMMENTS` → `transform.remove_comments`. (List-valued options like `skip_packages` are best set in YAML.)

## Examples

### Bundle to stdout and pipe straight into Lua

```bash
lua-amalgamate --entry src/main.lua --output - | lua -
```

### A directly executable program

```bash
lua-amalgamate --entry src/main.lua --shebang '#!/usr/bin/env lua' --output app
chmod +x app
./app
```

### Development build with accurate tracebacks

```bash
lua-amalgamate --entry src/main.lua --debug --output bundle.lua
```

A runtime error now points at the real source. Compare:

```
# without --debug
lua: bundle.lua:142: attempt to call a nil value

# with --debug
lua: src/lib/parser.lua:17: attempt to call a nil value
```

In debug mode each module is embedded as a string and run through `load(src, "@path")`, so the VM keeps the original chunk name and line numbers. Keep it **off for releases**: it adds a runtime dependency on `load`/`loadstring`, defeats `luac` precompilation, and is meaningless alongside line-altering transforms like `--minify`.

### Ship a bundle but override a module locally

```bash
lua-amalgamate --entry src/main.lua --fallback --output bundle.lua

# Run with a patched module on the path — the on-disk copy wins:
LUA_PATH="./patches/?.lua;;" lua bundle.lua
```

In fallback mode loaders live in `package.postload` behind a searcher appended *after* the path searchers, so anything found on `package.path` takes precedence. Without `--fallback`, the embedded copy always wins.

### Minified release build

```bash
lua-amalgamate --entry src/main.lua --minify --output dist/bundle.lua
```

`--minify` subsumes comment, empty-line, and shebang removal.

### Inject startup / teardown code

```bash
lua-amalgamate --entry src/main.lua \
  --prefix 'require("strict").on()' \
  --suffix 'print("bundle loaded")' \
  --output bundle.lua
```

### Package a library so it resolves to a flat file layout

Modules are named `mypkg.*` but files live directly under `src/` (no `mypkg/` directory):

```bash
lua-amalgamate --entry src/init.lua --package-name mypkg --output mypkg.lua
```

`--package-name mypkg` strips `mypkg.` while resolving (`require("mypkg.utils")` → `src/utils.lua`) and re-adds it to the output names, so the bundle is usable as `require("mypkg.utils")`.

### Exclude external dependencies

Leave system/3rd-party modules as ordinary runtime requires instead of embedding them:

```bash
lua-amalgamate --entry src/main.lua \
  --skip cjson --skip socket --skip 'xlog.*' \
  --output bundle.lua
```

### Fail the build on a missing module (CI)

```bash
lua-amalgamate --entry src/main.lua --strict --output /dev/null
```

With `--strict`, an unresolved static `require()` is an error (non-zero exit) instead of a warning.

### Search additional directories

```bash
lua-amalgamate --entry src/main.lua --search lib --search vendor --output bundle.lua
```

### Use a config file, override one value on the CLI

```bash
lua-amalgamate --config build/amalg.yaml --output /tmp/bundle.lua
```

## Use as a Go library

The bundler is exposed as a public package for embedding in other tools; the CLI
is a thin wrapper over it. Import the module root (package `amalgamate`):

```go
import (
	"bytes"
	"fmt"

	amalgamate "github.com/bigbes/lua-amalgamate"
)

func build() error {
	opts := amalgamate.DefaultOptions() // sensible defaults (arg-fix on, etc.)
	opts.Entry = "src/main.lua"
	opts.Debug = true // any option is a plain struct field

	var buf bytes.Buffer
	res, err := amalgamate.Bundle(opts, &buf)
	if err != nil {
		return err
	}
	for _, w := range res.Warnings {
		fmt.Printf("warning: %s:%d: %s\n", w.File, w.Line, w.Message)
	}
	// buf now holds the bundle.
	return nil
}
```

Public API:

- `amalgamate.Options` — all bundling settings (entry, root, path, transforms, `Debug`, `Fallback`, `Shebang`, `ArgFix`, prefixes, skip/include, …). `Bundle` writes to the `io.Writer` you give it and ignores `Options.Output`.
- `amalgamate.DefaultOptions() Options` — defaults matching the CLI.
- `amalgamate.LoadOptions(path string) (Options, error)` — load from a YAML file plus `AMALG_*` env vars (a missing file is not an error).
- `amalgamate.Bundle(opts Options, w io.Writer) (*Result, error)` — runs the full pipeline; resolves the root and the `package_name` convenience itself.
- `amalgamate.Result{ Warnings []Warning }` — non-fatal issues (dynamic/unresolved requires). In strict mode an unresolved require is a returned error instead.

Implementation packages live under `internal/` and are intentionally not importable; build against the facade above.

## Module resolution

`require("foo.bar")` is resolved like Lua's `package.searchpath`:

1. Dots become the path separator (`foo/bar`).
2. Each search location is tried in order: the entry's directory, `root`, then each `--search` directory.
3. At each location, every template in `--path` is applied (`?.lua`, then `?/init.lua`, …).
4. The first matching file wins.

Relative/path-style requires (`require("./foo")`, `require("../bar")`) are supported.

### Prefix stripping

For projects whose module names carry a package prefix that isn't mirrored in the directory layout:

- **`--strip-prefix tuple_diff`** — strips `tuple_diff.` before resolution, so `require("tuple_diff.lib.foo")` finds `lib/foo.lua`.
- **Auto-detection** — if the root directory name contains hyphens (e.g. `tuple-diff`), it's converted to `tuple_diff` and used as a strip prefix automatically.
- **Search-directory prefix** — when a module is found under `lib/`, the `lib.` prefix is also stripped (`lib.config` → `config`).
- **`--package-name mypkg`** — strip *and* re-add the prefix, so a library bundles from a flat layout yet stays `require("mypkg.*")`-compatible.

## Output format

Normal (default):

```lua
-- Amalgamated by lua-amalgamate
-- Entry: main

do
local _ENV = _ENV
package.preload["mymod"] = function(...)
  local name = ...
  package.loaded[name] = true
  local arg = _G.arg
-- module source, verbatim
end
end
return require("main")
```

Debug (`--debug`) — bodies are loaded with their original chunk name:

```lua
do
local _ENV = _ENV
package.preload["mymod"] = function(...)
  local name = ...
  package.loaded[name] = true
  local arg = _G.arg
  return assert((loadstring or load)([[
-- module source, verbatim
]], "@src/mymod.lua"))(...)
end
end
```

Fallback (`--fallback`) — a searcher is appended and loaders go into `package.postload`:

```lua
package.postload = package.postload or {}
do
  local postload = package.postload
  local searchers = package.searchers or package.loaders
  searchers[#searchers+1] = function(mod)
    local loader = postload[mod]
    if loader == nil then
      return "\n\tno field package.postload['" .. mod .. "']"
    end
    return loader
  end
end
```

## Limitations

- **Dynamic requires** (`require(someVariable)`) are detected but can't be resolved statically — they produce a warning and are left as runtime requires.
- **C modules** (`.so`/`.dll`) are not embedded; they remain runtime requires.
- **Parsing is Lua 5.1 syntax** (via the gopher-lua parser); some Lua 5.2+ syntax may not parse. The *generated* bundle runs on 5.1–5.4 and LuaJIT.
- **No asset embedding** — only `.lua` modules are bundled.

## Development

See [AGENTS.md](AGENTS.md) for build commands, code style, and contribution guidelines. Run the test suite with `go test ./...`; the integration tests additionally execute generated bundles under a `lua` interpreter when one is on `PATH`.

## License

MIT
