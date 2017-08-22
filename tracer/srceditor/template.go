package srceditor

import (
	"text/template"
	"bytes"
)

const (
	DefaultImportName = "__goapptrace_tracer"
	DefaultImportPath = "github.com/yuuki0xff/goapptrace/tracer/client"
)

type TemplateData struct {
	ImportName string
	ImportPath string
	D          interface{} // extra data
}

type Template struct {
	data TemplateData
	t    *template.Template
}

var tmpl Template

func init() {
	tmpl.add("importStmt", `
		import {{.ImportName}} {{.ImportPath}}
	`)
	tmpl.add("funcStartStopStmt", `
		{{.ImportName}}.FuncStart()
		defer {{.ImportName}}.FuncEnd()
	`)
}

func (t *Template) init(importName, importPath string) {
	t.data.ImportName = importName
	if importName == "" {
		t.data.ImportName = DefaultImportName
	}

	t.data.ImportPath = importPath
	if importPath == "" {
		t.data.ImportPath = DefaultImportPath
	}

	if t.t == nil {
		t.t = &template.Template{}
	}
}

func (t *Template) add(name, tmplStr string) {
	t.t = template.Must(t.t.New(name).Parse(tmplStr))
}

func (t *Template) render(name string, data interface{}) []byte {
	var buf bytes.Buffer
	var d = t.data
	d.D = data
	t.t.ExecuteTemplate(&buf, name, d)
	return buf.Bytes()
}
