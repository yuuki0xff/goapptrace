package srceditor

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
)

// 既存のソースコードを編集し、トレース用のコードを追加する。
type CodeEditor struct {
	// trueなら、エクスポートされた関数にのみトレース用のコードを追加する。
	// falseなら、全ての関数に対してトレース用のコードを追加する。
	ExportedOnly bool
	// import名や変数名につけるprefix。既存の変数などと名前が衝突しないようにするために設定する。
	Prefix string

	// コード編を出力するテンプレートを指定する。
	// nilの場合、 CodeEditor.init()で初期化される。
	tmpl *Template

	// unit test用のオプション。このオプションを指定すると、CodeEditor.random()が常に指定した文字列を返すようになる。
	// unit testの実行中に、実行結果が常に同じにようにするために使用することを想定している。
	dontUseRandom string
}

// inFileにトレース用コードを追加し、outFileに書き出す。
// inFileの内容は変更されない。
func (ce *CodeEditor) EditFile(inFile, outFile string) error {
	src, err := ioutil.ReadFile(inFile)
	if err != nil {
		return err
	}

	info, err := os.Stat(inFile)
	if err != nil {
		return err
	}

	var newSrc []byte
	if newSrc, err = ce.edit(inFile, src); err != nil {
		return err
	}

	return ioutil.WriteFile(outFile, newSrc, info.Mode())
}

// fnameにトレース用のコードを追加する。
// 指定されたファイルは上書きされる。
func (ce *CodeEditor) EditFileOverwrite(fname string) error {
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

	return AtomicReadWrite(fname, edit)
}

func AtomicReadWrite(fname string, fn func(r io.Reader, w io.Writer) error) error {
	var ok bool
	r, err := os.Open(fname)
	if err != nil {
		return err
	}
	defer r.Close() // nolint: errcheck

	finfo, err := os.Stat(fname)
	if err != nil {
		return err
	}

	w, err := ioutil.TempFile(path.Dir(fname), "."+path.Base(fname)+".tmp.")
	if err != nil {
		return err
	}
	tmpfname := w.Name()
	defer func() {
		if !ok {
			// clean up a temporary file.
			os.Remove(tmpfname) // nolint: errcheck
		}
	}()
	if err = w.Chmod(finfo.Mode()); err != nil {
		return err
	}

	if err = fn(r, w); err != nil {
		w.Close() // nolint: errcheck
		// the original file was kept, and tmp file will be remove.
		return err
	}

	// the original file was atomically replaced by a tmp file.
	if err = w.Close(); err != nil {
		return err
	}
	if err = os.Rename(tmpfname, fname); err != nil {
		return err
	}
	ok = true
	return nil
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

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, fname, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	nl := NodeList{
		File:    f,
		OrigSrc: src,
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
			data := struct {
				EscapedFuncName string
			}{
				// function name + hex random number
				// init関数は、同じパッケージ内に同名の関数を複数定義できる。このような場合に
				// flagの変数名が重複してしまう問題を回避するため、乱数を末尾に追加する。
				EscapedFuncName: node.Name.Name + "_" + ce.random(),
			}
			// define flags to enable/disable tracing each function.
			nl.Add(&InsertNode{
				Pos: f.End(),
				Src: ce.tmpl.render("defineFuncTracingFlag", data),
			})

			if pkgName == "main" && node.Name.Name == "main" {
				nl.Add(&InsertNode{
					Pos: node.Body.Lbrace + 1, // "{"の直後に挿入
					Src: ce.tmpl.render("funcStartCloseStopStmt", data),
				})
			} else {
				nl.Add(&InsertNode{
					Pos: node.Body.Lbrace + 1, // "{"の直後に挿入
					Src: ce.tmpl.render("funcStartStopStmt", data),
				})
			}
		case *ast.FuncLit:
			if node.Body == nil {
				// node is non-Go function
				return true
			}
			wantImport = true
			data := struct {
				EscapedFuncName string
			}{
				// 匿名の関数には、仮の名前としてランダムな文字列を指定する。
				EscapedFuncName: "anonymousFunc_" + ce.random(),
			}
			nl.Add(&InsertNode{
				Pos: f.End(),
				Src: ce.tmpl.render("defineFuncTracingFlag", data),
			})
			nl.Add(&InsertNode{
				Pos: node.Body.Lbrace + 1, // "{"の直後に挿入
				Src: ce.tmpl.render("funcStartStopStmt", data),
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
				// node means of call os.Exit().
				// We replace the "os.Exit(code)" statement to "logger.CloseAndExit(code)".
				nl.Add(&DeleteNode{
					Pos: a.Pos(),
					End: b.End(),
				})
				nl.Add(&InsertNode{
					Pos: b.End(),
					Src: ce.tmpl.render("closeAndExit", nil),
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
func (ce *CodeEditor) random() string {
	if ce.dontUseRandom != "" {
		return ce.dontUseRandom
	}
	return fmt.Sprintf("%016x%016x", rand.Int63(), rand.Int63())
}
