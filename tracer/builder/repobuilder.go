package builder

import (
	"fmt"
	"go/build"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/pkg/errors"
	"github.com/termie/go-shutil"
	"github.com/yuuki0xff/goapptrace/tracer/srceditor"
)

const (
	runtimePatch = `
package runtime

// GoID returns the Goroutine ID.
func GoID() int64 {
	gp := getg()
	return gp.goid
}
`
)

// トレース用のコードを追加したレポジトリを構築する。
// 編集後のコードは、Gorootとgopathで指定したディレクトリの下に出力される。
// オリジナルのコードは改変しない。
type RepoBuilder struct {
	// トレース用コード追加済みのstandard packagesの出力先
	Goroot string
	// トレース用コード追加済みのnon-standard packagesの出力先
	Gopath string

	// これらのパッケージと、これらが依存しているパッケージには、トレース用のコードを追加しない
	IgnorePkgs    map[string]bool
	IgnoreStdPkgs bool

	Editor     srceditor.CodeEditor
	OutputFile string
}

func (b *RepoBuilder) EditAll(targets []string) error {
	ok, err := IsGofiles(targets)
	if err != nil {
		return err
	}
	if ok {
		return b.EditFiles(targets)
	} else {
		return b.EditPackages(targets)
	}
}

func (b *RepoBuilder) Init() error {
	// copy golang pkg directory
	pkgDir := path.Join("pkg")
	src := path.Join(runtime.GOROOT(), pkgDir)
	dest := path.Join(b.Goroot, pkgDir)
	if err := shutil.CopyTree(src, dest, nil); err != nil {
		return err
	}

	// copy cgo libraries
	// copy src directory
	srcDir := path.Join("src")
	src = path.Join(runtime.GOROOT(), srcDir)
	dest = path.Join(b.Goroot, srcDir)
	return shutil.CopyTree(src, dest, nil)
}

// 指定されたソースコードと依存しているパッケージに、トレース用コードを追加する。
func (b *RepoBuilder) EditFiles(gofiles []string) error {
	for _, gofile := range gofiles {
		pkgName, err := packageName(gofile)
		if err != nil {
			return err
		}
		if pkgName != "main" {
			return fmt.Errorf("can not build non-main package files")
		}
	}

	// gofilesが依存しているパッケージ一覧を取得する
	imper := RecursiveImporter{
		IgnorePkgs: b.IgnorePkgs,
	}
	for _, gofile := range gofiles {
		if err := imper.ImportFromFile(gofile); err != nil {
			return err
		}
	}

	// 除外スべきパッケージ一覧を取得する
	ignoreImper := RecursiveImporter{}
	for pkg := range b.IgnorePkgs {
		if err := ignoreImper.ImportFromPkg(pkg); err != nil {
			return err
		}
	}

	mainpkg := b.MainPkgDir()
	if err := os.MkdirAll(mainpkg, os.ModePerm); err != nil {
		return err
	}
	for _, gofile := range gofiles {
		outfile := path.Join(mainpkg, path.Base(gofile))
		err := b.Editor.EditFile(gofile, outfile)
		if err != nil {
			return err
		}
	}

	for imppath, pkg := range imper.Pkgs() {
		if ignoreImper.Pkgs()[imppath] != nil {
			// 循環インポートを防ぐために、b.IgnorePkgsが依存しているパッケージは編集しない。
			continue
		}
		if b.IgnoreStdPkgs && isStdPkg(pkg.ImportPath) {
			destDir := path.Join(b.Goroot, "src", pkg.ImportPath)

			log.Printf("copying %s => %s", pkg.Dir, destDir)
			if err := copyPkg(pkg, destDir); err != nil {
				return err
			}
		} else {
			if err := b.editPackage(pkg); err != nil {
				return err
			}
		}
	}

	// トレース用コードを追加できないが、ビルドに必要なパッケージをコピーする
	for _, pkg := range ignoreImper.Pkgs() {
		destDir := path.Join(b.Gopath, "src", pkg.ImportPath)
		if isStdPkg(pkg.ImportPath) {
			destDir = path.Join(b.Goroot, "src", pkg.ImportPath)
		}

		log.Printf("copying %s => %s", pkg.Dir, destDir)
		if err := copyPkg(pkg, destDir); err != nil {
			return err
		}
	}

	// runtimeにパッチを当てる
	runtimeDir := path.Join(b.Goroot, "src", "runtime")
	patchFileName := path.Join(runtimeDir, "goapptrace.go")
	if err := os.MkdirAll(runtimeDir, 0777); err != nil {
		return err
	}
	if err := ioutil.WriteFile(patchFileName, []byte(runtimePatch), 0666); err != nil {
		return err
	}
	return nil
}

// 指定されたパッケージとその依存に、トレース用コードを追加する。
func (b *RepoBuilder) EditPackages(pkgs []string) error {
	imper := RecursiveImporter{
		IgnorePkgs: b.IgnorePkgs,
	}
	for _, pkg := range pkgs {
		if err := imper.ImportFromPkg(pkg); err != nil {
			return err
		}
	}

	ignoreImper := RecursiveImporter{}
	for pkg := range b.IgnorePkgs {
		if err := ignoreImper.ImportFromPkg(pkg); err != nil {
			return err
		}
	}

	for imppath, pkg := range imper.Pkgs() {
		if ignoreImper.Pkgs()[imppath] != nil {
			// 循環インポートを防ぐために、b.IgnorePkgsが依存しているパッケージは編集しない。
			continue
		}
		if err := b.editPackage(pkg); err != nil {
			return err
		}
	}
	return nil
}

// 指定したパッケージにトレース用コードを追加する。
func (b *RepoBuilder) editPackage(pkg *build.Package) error {
	var dir string
	if isStdPkg(pkg.ImportPath) {
		dir = path.Join(b.Goroot, "src", pkg.ImportPath)
	} else {
		dir = path.Join(b.Gopath, "src", pkg.ImportPath)
	}
	log.Printf("editing %s package (stdpkg=%t) ... ", pkg.ImportPath, isStdPkg(pkg.ImportPath))
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	for _, gofile := range pkg.GoFiles {
		srcfile := path.Join(pkg.Dir, gofile)
		destfile := path.Join(dir, gofile)
		log.Printf("editing %s => %s", srcfile, destfile)
		if err := b.Editor.EditFile(srcfile, destfile); err != nil {
			return err
		}
	}
	return nil
}

func (b *RepoBuilder) MainPkgDir() string {
	return path.Join(b.Gopath, "mainpkg")
}

// 全てのファイルが".go"で終わるファイルなら、trueを返す
func IsGofiles(files []string) (bool, error) {
	for _, f := range files {
		if !strings.HasSuffix(f, ".go") {
			return false, nil
		}
		finfo, err := os.Stat(f)
		if err != nil {
			return false, errors.Wrap(err, "not found *.go file")
		}
		if finfo.IsDir() {
			return false, nil
		}
	}
	return true, nil
}

// gofileのパッケージ名を返す
func packageName(gofile string) (string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, gofile, nil, parser.ParseComments)
	if err != nil {
		return "", err
	}
	return f.Name.Name, nil
}

// copy all regular files under "pkg.Dir" directory to destDir.
func copyPkg(pkg *build.Package, destDir string) error {
	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		return err
	}

	finfos, err := ioutil.ReadDir(pkg.Dir)
	if err != nil {
		return err
	}
	files := make([]string, 0, len(finfos))
	for i := range finfos {
		if finfos[i].Mode().IsRegular() {
			files = append(files, finfos[i].Name())
		}
	}

	for _, gofile := range files {
		srcfile := path.Join(pkg.Dir, gofile)
		destfile := path.Join(destDir, gofile)

		if err := shutil.CopyFile(srcfile, destfile, false); err != nil {
			return err
		}
	}
	return nil
}
