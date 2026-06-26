package emit

import (
	"fmt"
	"strings"
	"text/template"
)

// Each module is wrapped following the lua-amalg pattern: `local _ENV = _ENV`
// lives in the enclosing `do` block (not the function body) so it becomes the
// loader's first upvalue, which Lua 5.2's `module`/`_ENV` handling relies on.
// `local arg = _G.arg` counters the implicit `arg` local that Lua 5.1
// (LUA_COMPAT_VARARG) injects into vararg functions. `package.loaded[name]` is
// set before the body runs to break circular requires. Aliases delegate to the
// primary name so the body executes exactly once regardless of how it's required.
//
// By default loaders go into package.preload, which the built-in preload
// searcher consults before searching package.path — so embedded modules win.
// In fallback mode (.Fallback) loaders go into package.postload and a searcher
// is appended last, so an on-disk copy on package.path takes precedence and the
// embedded module is used only as a fallback.
const templateText = `{{if .Shebang}}{{.Shebang}}
{{end}}-- Amalgamated by lua-amalgamate
-- Entry: {{.EntryName}}

{{if .Prefix}}{{.Prefix}}
{{end}}{{if .Fallback}}package.postload = package.postload or {}
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

{{end}}{{range .Modules}}{{$primary := .PrimaryName}}do
local _ENV = _ENV
package.{{$.RegTable}}[{{q $primary}}] = function(...)
  local name = ...
  package.loaded[name] = true
{{if not $.NoArgFix}}  local arg = _G.arg
{{end}}{{if $.Debug}}  return assert((loadstring or load)({{bracket .Source}}, {{q (chunkname .Path)}}))(...)
{{else}}{{indent .Source 2}}{{end}}end
end
{{range .AliasNames}}package.{{$.RegTable}}[{{q .}}] = function(...) return require({{q $primary}}) end
{{end}}{{end}}return require({{q .EntryName}}){{if .Suffix}}
{{.Suffix}}{{end}}
`

func indent(source string, spaces int) string {
	if source == "" {
		return ""
	}
	prefix := strings.Repeat(" ", spaces)
	lines := strings.Split(strings.TrimRight(source, "\n"), "\n")
	var out strings.Builder
	for _, line := range lines {
		out.WriteString(prefix)
		out.WriteString(line)
		out.WriteByte('\n')
	}
	return out.String()
}

// quoteLua renders s as a Lua double-quoted string literal, escaping any
// character that would otherwise break the literal (quotes, backslashes,
// newlines, control bytes). Module names come from file paths and are normally
// trivial, but this keeps the emitter robust against unusual names.
func quoteLua(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		default:
			if c < 0x20 || c == 0x7f {
				fmt.Fprintf(&b, `\%d`, c)
			} else {
				b.WriteByte(c)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

// luaLongBracket wraps s in a Lua long-bracket string literal (`[==[ … ]==]`),
// choosing a level of `=` that does not collide with any closing sequence in s.
// A newline is inserted right after the opening bracket; Lua discards that first
// newline, so the embedded source keeps its original line numbering — which is
// the whole point in debug mode. The source is embedded verbatim (no escaping,
// no reindentation), so multi-line string literals inside modules survive intact.
func luaLongBracket(s string) string {
	level := 0
	for strings.Contains(s, "]"+strings.Repeat("=", level)+"]") {
		level++
	}
	eq := strings.Repeat("=", level)
	return "[" + eq + "[\n" + s + "]" + eq + "]"
}

// chunkName renders the load() chunk name for a module path. The leading '@'
// tells Lua to treat the rest as a file name in error messages and tracebacks.
func chunkName(path string) string {
	return "@" + path
}

var outputTemplate = template.Must(
	template.New("output").Funcs(template.FuncMap{
		"indent":    indent,
		"q":         quoteLua,
		"bracket":   luaLongBracket,
		"chunkname": chunkName,
	}).Parse(templateText),
)

type templateData struct {
	EntryName string
	Modules   []moduleData
	Prefix    string
	Suffix    string
	Shebang   string
	Debug     bool
	Fallback  bool
	NoArgFix  bool
	// RegTable is the package field loaders are registered in: "preload"
	// normally, "postload" in fallback mode.
	RegTable string
}

type moduleData struct {
	PrimaryName string
	AliasNames  []string
	Source      string
	Path        string
}
