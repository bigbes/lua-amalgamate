# Amalgamator - Agent Guidelines

## Overview
Go project (Go 1.25.1) implementing a Lua amalgamator (bundler). Merges multi‑file Lua projects into a single self‑contained `.lua` file while preserving exact `require()` semantics.

Design: `plan.ai.md`. Implementation status: `todo.ai.md`.

**Recent features**: shebang stripping (`--strip‑shebang`), prefix/suffix code injection (`--prefix`, `--suffix`), package prefix (`--package‑prefix`), package name (`--package‑name`), skip packages (`--skip`), include packages (`--include`), circular dependency fix, strip prefix (`--strip‑prefix`), integration test suite.

## Build & Test Commands

### Building
```bash
go build ./...                    # All packages
go build ./cmd/lua-amalgamate     # CLI tool
go install ./cmd/lua-amalgamate   # Install globally
```

### Testing
```bash
go test ./...                     # All tests
go test ./internal/config         # Specific package
go test ./internal/integration    # Integration tests (requires Lua interpreter)
go test -run TestLoadConfig ./internal/config  # Single test
go test -v ./...                  # Verbose
go test -coverprofile=coverage.out ./...       # Coverage
go tool cover -html=coverage.out
```

### Linting & Quality
```bash
go vet ./...                      # Vet for suspicious constructs
gofmt -d .                        # Check formatting (dry‑run)
gofmt -w .                        # Apply formatting
golangci-lint run                 # If installed
```

### Running
```bash
go run ./cmd/lua-amalgamate --entry main.lua
go run ./cmd/lua-amalgamate --config amalg.yaml
./lua-amalgamate --entry main.lua # After build

# New features examples:
./lua-amalgamate --entry main.lua --strip-shebang
./lua-amalgamate --entry main.lua --prefix "print('start')" --suffix "print('end')"
./lua-amalgamate --entry main.lua --strip-prefix "tuple_diff"
./lua-amalgamate --entry main.lua --package-name "mylib"
```

### Releasing
```bash
# Install goreleaser
go install github.com/goreleaser/goreleaser/v2@latest

# Dry run (snapshot)
goreleaser release --snapshot --clean

# Create a new tag and push, then release (requires GITHUB_TOKEN)
git tag v1.0.0
git push origin v1.0.0
# GitHub Actions will automatically run goreleaser
```

### Docker Images
GoReleaser builds multi‑platform Docker images (linux/amd64, linux/arm64) and publishes them to GitHub Container Registry:

```bash
# Build and push Docker images (requires DOCKER_USERNAME/DOCKER_PASSWORD or GITHUB_TOKEN)
goreleaser release --clean

# Skip Docker builds during snapshot
goreleaser release --snapshot --clean --skip=docker

# Images available at:
# ghcr.io/bigbes/lua-amalgamate:latest
# ghcr.io/bigbes/lua-amalgamate:vX.Y.Z
```

See `.goreleaser.yml` for configuration.

## Code Style Guidelines

### Formatting & Naming
- Use **gofmt**. Line length ≤100 chars. Tabs for indentation.
- **Exported**: `PascalCase`. **Unexported**: `camelCase`.
- **Acronyms**: whole words (`HTTPClient` not `HttpClient`).
- **Interfaces**: single‑method end with `‑er` (`Parser`); multi‑method use descriptive nouns.
- **Variables**: short, descriptive; `i`, `j` for loops, `err` for errors.
- **Constants**: `UPPER_SNAKE_CASE` (exported); unexported may use `camelCase`.
- **Files**: lowercase with underscores only when necessary (`config.go`, `gopherparse.go`).

### Imports
Group in order, separated by blank line:
1. Standard library.
2. Third‑party packages.
3. Internal packages.

Example:
```go
import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/yuin/gopher-lua/ast"
    "github.com/yuin/gopher-lua/parse"
    "gopkg.in/yaml.v3"

    "github.com/bigbes/lua-amalgamate/internal/config"
)
```
Use import aliases only to avoid conflicts.

### Types
- **Structs**: define with YAML struct tags for config fields (`` `yaml:"entry"` ``).
- **Interfaces**: keep small and focused (`Parser`, `Transformer`).
- **Methods**: value receivers unless modifying receiver.
- **Zero values**: design types so zero value is useful.

### Error Handling
- Error messages: start with lowercase, no trailing punctuation.
- Wrap errors with context: `fmt.Errorf("load config: %w", err)`.
- Avoid panics in library code; panic only for unrecoverable programmer errors.
- Custom error types if callers need to inspect details.
- Exported error variables: `ErrX` (e.g., `ErrNoEntry`).

### Logging & Debugging
- CLI uses `fmt.Fprintf(os.Stderr, ...)` for user‑facing messages.
- Warnings collected during graph building, printed to stderr after emission.
- Debug output: use `--verbose` flag if added; otherwise no debug logging.

### Testing
- Test files: `*_test.go` in same package.
- Table‑driven tests encouraged.
- Test helpers: unexported functions prefixed with `test`.
- Golden files in `testdata/` for integration tests.
- Mocking: interface‑based fakes (e.g., `mockParser`).
- Coverage: aim high for core logic; 100% not required for trivial code.

## Project Structure
```
lua-amalgamate/
├── cmd/lua-amalgamate/          # CLI entry point
├── internal/
│   ├── config/                   # YAML config + CLI flag merging
│   ├── parse/                    # Parser interface + gopher‑lua implementation
│   ├── resolve/                  # Lua‑style module path resolution
│   ├── graph/                    # BFS dependency graph builder
│   ├── emit/                     # Single‑file Lua emitter
│   ├── transform/                # Source transformations
│   └── integration/              # End‑to‑end integration tests
├── testdata/                     # Test fixtures (Lua projects)
├── amalg.yaml.example            # Example config file
├── README.md                     # Documentation
├── plan.ai.md                    # Detailed design document
├── todo.ai.md                    # Implementation todo list
└── go.mod
```

## Dependencies
- `github.com/yuin/gopher‑lua` – only `parse` and `ast` sub‑packages (no VM).
- `gopkg.in/yaml.v3` – YAML config parsing.
- Standard library – `flag`, `io`, `os`, `path/filepath`, etc.

Keep dependency graph small; add only when necessary.

## Development Workflow
1. **Implement core functionality first** (phases in `todo.ai.md`).
2. **Write tests** for each package as you go.
3. **Run `go vet` and `gofmt`** before committing.
4. **Verify integration** by running amalgamator on `testdata/` projects.
5. **Add transformations** after core is stable.

## Cursor / Copilot Rules
No project‑specific rules exist. Follow guidelines above.

## Commit Messages
- Imperative mood ("Add feature", not "Added feature").
- First line ≤50 chars, blank line, then body (if needed).
- Reference issues or design documents when appropriate.

## Pull Requests
- Include relevant unit and integration tests.
- Ensure `go test ./...` passes.
- Update `AGENTS.md` if introducing new conventions.

---

*Last updated: 2026‑02‑17*