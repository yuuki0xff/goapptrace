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
	tc.Editor.dontUseRandom = "random"
	tc.Editor.tmpl = newTestTemplate(TemplateData{})

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

import __goapptrace_tracer "github.com/yuuki0xff/goapptrace/tracer/logger"

func ExportedFunc(a, b, c string) stirng {
	/* startStop(ExportedFunc_random) */

	return "ok"
}

func nonExportedFunc() string {
	/* startStop(nonExportedFunc_random) */

	return "ok"
}

/* defineVar(ExportedFunc_random) */

/* defineVar(nonExportedFunc_random) */
`),
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

import __goapptrace_tracer "github.com/yuuki0xff/goapptrace/tracer/logger"

func ExportedFunc(a, b, c string) stirng {
	/* startStop(ExportedFunc_random) */

	return "ok"
}

func nonExportedFunc() string {
	return "ok"
}

/* defineVar(ExportedFunc_random) */
`),
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

import __goapptrace_tracer "github.com/yuuki0xff/goapptrace/tracer/logger"

var ExportedVar = func() string {
	/* startStop(anonymousFunc_random) */
	return "ok"
}
var nonExportedVar = func() string {
	/* startStop(anonymousFunc_random) */
	return "ok"
}

func ExportedFunc() {
	/* startStop(ExportedFunc_random) */

	fn := func() string {
		/* startStop(anonymousFunc_random) */

		return "in function"
	}

	go func() string {
		/* startStop(anonymousFunc_random) */

		return "in go statement"
	}()

	caller(func() string {
		/* startStop(anonymousFunc_random) */

		go func() string {
			/* startStop(anonymousFunc_random) */

			return "nested"
		}()
		return "in call statement"
	})
}

/* defineVar(anonymousFunc_random) */

/* defineVar(anonymousFunc_random) */

/* defineVar(ExportedFunc_random) */

/* defineVar(anonymousFunc_random) */

/* defineVar(anonymousFunc_random) */

/* defineVar(anonymousFunc_random) */

/* defineVar(anonymousFunc_random) */
`),
	})
}

func TestEditMainFunc(t *testing.T) {
	testEdit(t, editTestCase{
		Editor: CodeEditor{},
		In: strings.TrimSpace(`
package main

import "fmt"

func main() {
	// comment
	fmt.Println("Hello World!")
}
`),
		Out: strings.TrimSpace(`
package main

import __goapptrace_tracer "github.com/yuuki0xff/goapptrace/tracer/logger"

import "fmt"

func main() {
	/* startCloseStop(main_random) */

	// comment
	fmt.Println("Hello World!")
}

/* defineVar(main_random) */
`),
	})
}

func TestEditMainFuncInNonMainPackage(t *testing.T) {
	testEdit(t, editTestCase{
		Editor: CodeEditor{},
		In: strings.TrimSpace(`
package hoge

import "fmt"

func main() {
	// comment
	fmt.Println("Hello World!")
}
`),
		Out: strings.TrimSpace(`
package hoge

import __goapptrace_tracer "github.com/yuuki0xff/goapptrace/tracer/logger"

import "fmt"

func main() {
	/* startStop(main_random) */

	// comment
	fmt.Println("Hello World!")
}

/* defineVar(main_random) */
`),
	})
}

func TestEditCallOsExit(t *testing.T) {
	testEdit(t, editTestCase{
		Editor: CodeEditor{},
		In: strings.TrimSpace(`
package foo

import "fmt"
import "os"

func bar() {
	fmt.Println("Hello World!")
	os.Exit(0)
}
`),
		Out: strings.TrimSpace(`
package foo

import __goapptrace_tracer "github.com/yuuki0xff/goapptrace/tracer/logger"

import "fmt"
import "os"

func bar() {
	/* startStop(bar_random) */

	fmt.Println("Hello World!")
	closeAndExit(0)
}

/* defineVar(bar_random) */
`),
	})
}

func TestEditIncludeCommentsBeforePackageStatement(t *testing.T) {
	testEdit(t, editTestCase{
		Editor: CodeEditor{},
		In: strings.TrimSpace(`
// Copyright 2018 yuuki0xff.

package foo

import "os"

func bar() {
	os.Exit(0)
}
`),
		Out: strings.TrimSpace(`
// Copyright 2018 yuuki0xff.

package foo

import __goapptrace_tracer "github.com/yuuki0xff/goapptrace/tracer/logger"

import "os"

func bar() {
	/* startStop(bar_random) */

	closeAndExit(0)
}

/* defineVar(bar_random) */
`),
	})
}
