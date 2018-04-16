package srceditor

import (
	"bytes"
	"text/template"
)

const (
	DefaultImportName     = "__goapptrace_tracer"
	DefaultImportPath     = "github.com/yuuki0xff/goapptrace/tracer/logger"
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
	t := &Template{
		data: data,
		t:    &template.Template{},
	}

	if t.data.ImportName == "" {
		t.data.ImportName = DefaultImportName
	}

	if t.data.ImportPath == "" {
		t.data.ImportPath = DefaultImportPath
	}

	if t.data.VariablePrefix == "" {
		t.data.VariablePrefix = DefaultVariablePrefix
	}

	t.add("_funcTracingFlag", "{{.VariablePrefix}}_func_{{.D.EscapedFuncName}}_isTracing")

	// "package"宣言の次の行に挿入される。
	t.add("importStmt", `
		import {{.ImportName}} "{{.ImportPath}}"
	`)
	// トレース機能が有効化されているか識別するためのフラグを定義する。
	// "import"ステートメントの次の行に挿入される。
	t.add("defineFuncTracingFlag", `
		var {{template "_funcTracingFlag" .}} *bool
	`)
	// 関数の"{"の直後に挿入される。
	// formatすると、関数の最初の行でFuncStart()を呼び出すようになる。
	// "{"の後ろで開業されている場合、オリジナルのコードとの間には1行の空白が存在するはずである。
	t.add("funcStartStopStmt", `
		if {{template "_funcTracingFlag" .}} == nil {
			{{template "_funcTracingFlag" .}} = {{.ImportName}}.TracingFlag()
		}
		if *{{template "_funcTracingFlag" .}} {
			{{.VariablePrefix}}_txid := {{.ImportName}}.FuncStart()
			defer {{.ImportName}}.FuncEnd({{.VariablePrefix}}_txid)
		}
	`)
	// "funcStartStopStmt"と同様。ただし、mainパッケージのmain関数のみに適用される。
	t.add("funcStartCloseStopStmt", `
		defer {{.ImportName}}.Close()
		{{template "funcStartStopStmt" .}}
	`)
	// os.Exit()の呼び出しを行う直前の行に挿入される。
	t.add("closeAndExit", "{{.ImportName}}.CloseAndExit")
	return t
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
