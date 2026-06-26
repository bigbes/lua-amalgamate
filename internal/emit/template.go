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
const templateText = `
-- Amalgamated by lua-amalgamate
-- Entry: {{.EntryName}}

{{if .Prefix}}{{.Prefix}}{{end}}
{{range .Modules}}{{$primary := .PrimaryName}}do
local _ENV = _ENV
package.preload[{{q $primary}}] = function(...)
  local name = ...
  package.loaded[name] = true
  local arg = _G.arg
{{indent .Source 2}}end
end
{{range .AliasNames}}package.preload[{{q .}}] = function(...) return require({{q $primary}}) end
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

var outputTemplate = template.Must(
	template.New("output").Funcs(template.FuncMap{
		"indent": indent,
		"q":      quoteLua,
	}).Parse(templateText),
)

type templateData struct {
	EntryName string
	Modules   []moduleData
	Prefix    string
	Suffix    string
}

type moduleData struct {
	PrimaryName string
	AliasNames  []string
	Source      string
}
