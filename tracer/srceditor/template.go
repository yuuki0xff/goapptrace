package srceditor

import (
	"bytes"
	"text/template"
)

const (
	DefaultImportName     = "__goapptrace_tracer"
	DefaultImportPath     = "github.com/yuuki0xff/goapptrace/tracer/client"
	DefaultVariablePrefix = "__goapptrace_tracer_var_"
)

type TemplateData struct {
	ImportName     string
	ImportPath     string
	VariablePrefix string
	D              interface{} // extra data
}

type Template struct {
	data TemplateData
	t    *template.Template
}

func newTemplate(data TemplateData) *Template {
	var t Template

	t.data = data
	if t.data.ImportName == "" {
		t.data.ImportName = DefaultImportName
	}

	if t.data.ImportPath == "" {
		t.data.ImportPath = DefaultImportPath
	}

	if t.data.VariablePrefix == "" {
		t.data.VariablePrefix = DefaultVariablePrefix
	}

	if t.t == nil {
		t.t = &template.Template{}
	}

	t.add("importStmt", `
		import {{.ImportName}} "{{.ImportPath}}"
	`)
	t.add("funcStartStopStmt", `
		{{.ImportName}}.FuncStart()
		defer {{.ImportName}}.FuncEnd()
	`)
	return &t
}

func (t *Template) add(name, tmplStr string) {
	t.t = template.Must(t.t.New(name).Parse(tmplStr))
}

func (t *Template) render(name string, data interface{}) []byte {
	var buf bytes.Buffer
	var d = t.data
	d.D = data
	if err := t.t.ExecuteTemplate(&buf, name, d); err != nil {
		panic(err)
	}
	return buf.Bytes()
}
