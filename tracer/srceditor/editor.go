package srceditor

import (
	"io/ioutil"
	"io"
	"os"
	"path"
	"go/token"
	"go/parser"
	"go/ast"
	"unicode"
)

type CodeEditor struct {
	ExportedOnly bool
	Prefix       string
	Files        []string
	Overwrite    bool
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
	if err := AtomicReadWrite(fname, func(r io.Reader, w io.Writer) error {
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
	}); err != nil {
		return err
	}
	return nil
}

func AtomicReadWrite(fname string, fn func(r io.Reader, w io.Writer) error) error {
	r, err := os.Open(fname)
	if err != nil {
		return err
	}
	defer r.Close()

	tmpfname := path.Join(path.Dir(fname), "."+path.Base(fname)+".tmp")
	w, err := os.OpenFile(tmpfname, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0)
	if err != nil {
		return err
	}
	defer w.Close()

	if err = fn(r, w); err != nil {
		// the original file was kept, and tmp file will be remove.
		os.Remove(tmpfname)
		return err
	}

	// the original file was atomically replaced by a tmp file.
	if err := w.Close(); err != nil {
		return err
	}
	return os.Rename(tmpfname, fname)
}

func (ce *CodeEditor) edit(fname string, src []byte) ([]byte, error) {
	nl := NodeList{OrigSrc: src}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, fname, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// insert tracing code into functions
	ast.Inspect(f, func(node_ ast.Node) bool {
		switch node := node_.(type) {
		case *ast.FuncDecl:
			if ce.ExportedOnly && !isExported(node.Name.Name) {
				// do not enter into function
				return false
			}

			nl.Add(&InsertNode{
				Pos: node.Body.Pos(),
				Src: tmpl.render("funcStartStopStmt", nil),
			})
		case *ast.FuncLit:
			nl.Add(&InsertNode{
				Pos: node.Body.Pos(),
				Src: tmpl.render("funcStartStopStmt", nil),
			})
		}
		return true
	})

	return nl.Format()
}

func isExported(funcname string) bool {
	for _, rune := range funcname {
		return unicode.IsUpper(rune)
	}
	panic("Unreachable")
}
