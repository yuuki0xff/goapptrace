package srceditor

import (
	"strings"
	"testing"

	"github.com/pmezard/go-difflib/difflib"
)

type editTestCase struct {
	Editor CodeEditor
	In     string
	Out    string
}

func testEdit(t *testing.T, tc editTestCase) {
	outbytes, err := tc.Editor.edit("test.go", []byte(tc.In))
	if err != nil {
		t.Error(err)
		return
	}
	out := strings.TrimSpace(string(outbytes))
	if out != tc.Out {
		diff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
			A:        difflib.SplitLines(tc.Out),
			B:        difflib.SplitLines(out),
			FromFile: "expected.go",
			ToFile:   "output.go",
			Context:  5,
		})
		if err != nil {
			panic(err)
		}

		t.Errorf(`Got unexpected output
Input
=====
%s

Output Diff
======
%s`, tc.In, diff)
		return
	}
}

func TestEditNoFunc(t *testing.T) {
	testEdit(t, editTestCase{
		Editor: CodeEditor{},
		In: strings.TrimSpace(`
package example

const ThisIsConst = true

var ThisIsVariable bool
`),
		Out: strings.TrimSpace(`
package example

const ThisIsConst = true

var ThisIsVariable bool
`),
	})
}

func TestEditAllFunc(t *testing.T) {
	testEdit(t, editTestCase{
		Editor: CodeEditor{},
		In: strings.TrimSpace(`
package example

func ExportedFunc(a, b, c string) stirng {
	return "ok"
}

func nonExportedFunc() string {
	return "ok"
}`),
		Out: strings.TrimSpace(`
package example

import __goapptrace_tracer "github.com/yuuki0xff/goapptrace/tracer/client"

func ExportedFunc(a, b, c string) stirng {
	__goapptrace_tracer.FuncStart()
	defer __goapptrace_tracer.FuncEnd()

	return "ok"
}

func nonExportedFunc() string {
	__goapptrace_tracer.FuncStart()
	defer __goapptrace_tracer.FuncEnd()

	return "ok"
}`),
	})
}

func TestEditExportedOnly(t *testing.T) {
	testEdit(t, editTestCase{
		Editor: CodeEditor{
			ExportedOnly: true,
		},
		In: strings.TrimSpace(`
package example

func ExportedFunc(a, b, c string) stirng{
	return "ok"
}

func nonExportedFunc() string {
	return "ok"
}`),
		Out: strings.TrimSpace(`
package example

import __goapptrace_tracer "github.com/yuuki0xff/goapptrace/tracer/client"

func ExportedFunc(a, b, c string) stirng {
	__goapptrace_tracer.FuncStart()
	defer __goapptrace_tracer.FuncEnd()

	return "ok"
}

func nonExportedFunc() string {
	return "ok"
}`),
	})
}

func TestEditFuncStmt(t *testing.T) {
	testEdit(t, editTestCase{
		Editor: CodeEditor{},
		In: strings.TrimSpace(`
package example

var ExportedVar = func() string { return "ok" }
var nonExportedVar = func() string { return "ok" }

func ExportedFunc() {
	fn := func () string {
		return "in function"
	}

	go func () string {
		return "in go statement"
	}()

	caller(func () string {
		go func () string {
			return "nested"
		}()
		return "in call statement"
	})
}
`),
		Out: strings.TrimSpace(`
package example

import __goapptrace_tracer "github.com/yuuki0xff/goapptrace/tracer/client"

var ExportedVar = func() string {
	__goapptrace_tracer.FuncStart()
	defer __goapptrace_tracer.FuncEnd()
	return "ok"
}
var nonExportedVar = func() string {
	__goapptrace_tracer.FuncStart()
	defer __goapptrace_tracer.FuncEnd()
	return "ok"
}

func ExportedFunc() {
	__goapptrace_tracer.FuncStart()
	defer __goapptrace_tracer.FuncEnd()

	fn := func() string {
		__goapptrace_tracer.FuncStart()
		defer __goapptrace_tracer.FuncEnd()

		return "in function"
	}

	go func() string {
		__goapptrace_tracer.FuncStart()
		defer __goapptrace_tracer.FuncEnd()

		return "in go statement"
	}()

	caller(func() string {
		__goapptrace_tracer.FuncStart()
		defer __goapptrace_tracer.FuncEnd()

		go func() string {
			__goapptrace_tracer.FuncStart()
			defer __goapptrace_tracer.FuncEnd()

			return "nested"
		}()
		return "in call statement"
	})
}
`),
	})
}
