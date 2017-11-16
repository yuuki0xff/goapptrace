package srceditor

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"path"
)

type CodeEditor struct {
	ExportedOnly bool
	Prefix       string
	Files        []string
	Overwrite    bool

	tmpl *Template
}

func (ce *CodeEditor) EditAll() error {
	for _, f := range ce.Files {
		if err := ce.Edit(f); err != nil {
			return err
		}
	}
	return nil
}

func (ce *CodeEditor) Edit(fname string) error {
	edit := func(r io.Reader, w io.Writer) error {
		src, err := ioutil.ReadAll(r)
		if err != nil {
			return err
		}

		var newSrc []byte
		if newSrc, err = ce.edit(fname, src); err != nil {
			return err
		}

		_, err = w.Write(newSrc)
		return err
	}

	if ce.Overwrite {
		return AtomicReadWrite(fname, edit)
	} else {
		file, err := os.Open(fname)
		if err != nil {
			return err
		}
		defer file.Close() // nolint: errcheck
		return edit(file, os.Stdout)
	}
}

func AtomicReadWrite(fname string, fn func(r io.Reader, w io.Writer) error) error {
	r, err := os.Open(fname)
	if err != nil {
		return err
	}
	defer r.Close() // nolint: errcheck

	finfo, err := os.Stat(fname)
	if err != nil {
		return err
	}

	tmpfname := path.Join(path.Dir(fname), "."+path.Base(fname)+".tmp")
	w, err := os.OpenFile(tmpfname, os.O_CREATE|os.O_WRONLY|os.O_EXCL, finfo.Mode())
	if err != nil {
		return err
	}
	defer w.Close() // nolint: errcheck

	if err = fn(r, w); err != nil {
		// the original file was kept, and tmp file will be remove.
		if err := os.Remove(tmpfname); err != nil {
			return err
		}
		return err
	}

	// the original file was atomically replaced by a tmp file.
	if err := w.Close(); err != nil {
		return err
	}
	return os.Rename(tmpfname, fname)
}

func (ce *CodeEditor) init() {
	if ce.tmpl == nil {
		var importPrefix, varPrefix string
		if ce.Prefix != "" {
			importPrefix = ce.Prefix + "_import"
			varPrefix = ce.Prefix + "_var_"
		}

		ce.tmpl = newTemplate(TemplateData{
			ImportName:     importPrefix,
			VariablePrefix: varPrefix,
		})
	}
}

func (ce *CodeEditor) edit(fname string, src []byte) ([]byte, error) {
	ce.init()

	nl := NodeList{OrigSrc: src}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, fname, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// insert tracing code into functions
	pkgName := f.Name.Name
	var wantImport bool
	ast.Inspect(f, func(node_ ast.Node) bool {
		switch node := node_.(type) {
		case *ast.FuncDecl:
			if ce.ExportedOnly && !node.Name.IsExported() {
				// do not enter into function
				return false
			}
			if node.Body == nil {
				// node is non-Go function
				return true
			}
			wantImport = true
			if pkgName == "main" && node.Name.Name == "main" {
				nl.Add(&InsertNode{
					Pos: node.Body.Pos(),
					Src: ce.tmpl.render("funcStartCloseStopStmt", nil),
				})
			} else {
				nl.Add(&InsertNode{
					Pos: node.Body.Pos(),
					Src: ce.tmpl.render("funcStartStopStmt", nil),
				})
			}
		case *ast.FuncLit:
			if node.Body == nil {
				// node is non-Go function
				return true
			}
			wantImport = true
			nl.Add(&InsertNode{
				Pos: node.Body.Pos(),
				Src: ce.tmpl.render("funcStartStopStmt", nil),
			})
		case *ast.CallExpr:
			selNode, ok := node.Fun.(*ast.SelectorExpr)
			if !ok {
				break
			}
			a, ok := selNode.X.(*ast.Ident)
			if !ok {
				break
			}
			b := selNode.Sel
			if a.Name == "os" && b.Name == "Exit" {
				// node means of call os.Exit()
				nl.Add(&InsertNode{
					Pos: a.Pos(),
					Src: ce.tmpl.render("close", node),
				})
			}
		}
		return true
	})

	// insert a import statement after package statement
	if wantImport {
		nl.Add(&InsertNode{
			Pos: f.Name.End(),
			Src: ce.tmpl.render("importStmt", nil),
		})
	}

	return nl.Format()
}
