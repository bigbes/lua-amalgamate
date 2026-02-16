package emit

import (
	"strings"
	"text/template"
)

const templateText = `
-- Amalgamated by lua-amalgamate
-- Entry: {{.EntryName}}

{{if .Prefix}}{{.Prefix}}{{end}}
{{range .Modules}}{{if .IsSingle}}package.preload["{{.PrimaryName}}"] = function(...)
 local name = ...
 package.loaded[name] = true
{{indent .Source 2}}end
{{else}}do
 local __loader = function(...)
   local name = ...
   package.loaded[name] = true
{{indent .Source 2}}  end{{range .Names}}
 package.preload["{{.}}"] = __loader{{end}}
end
{{end}}
{{end}}return require("{{.EntryName}}"){{if .Suffix}}
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

var outputTemplate = template.Must(
	template.New("output").Funcs(template.FuncMap{
		"indent": indent,
	}).Parse(templateText),
)

type templateData struct {
	EntryName string
	Modules   []moduleData
	Prefix    string
	Suffix    string
}

type moduleData struct {
	Names       []string
	Source      string
	IsSingle    bool
	PrimaryName string
}
