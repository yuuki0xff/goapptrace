package srceditor

import "text/template"

func newTestTemplate(data TemplateData) *Template {
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
		t.data.VariablePrefix = "var_"
	}

	t.add("_funcTracingFlag", "{{.VariablePrefix}}_func_{{.D.EscapedFuncName}}_isTracing")

	t.add("importStmt", `
		import {{.ImportName}} "{{.ImportPath}}"
	`)
	t.add("defineFuncTracingFlag", `
		/* defineVar({{.D.EscapedFuncName}}) */
	`)
	t.add("funcStartStopStmt", `
		/* startStop({{.D.EscapedFuncName}}) */
	`)
	t.add("funcStartCloseStopStmt", `
		/* startCloseStop({{.D.EscapedFuncName}}) */
	`)
	t.add("closeAndExit", "closeAndExit")
	return t
}
