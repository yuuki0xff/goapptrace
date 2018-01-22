package builder

import (
	"go/build"
	"go/parser"
	"go/token"
	"log"
	"path/filepath"
	"strings"
)

// 依存関係のあるパッケージを全てインポートする。
type RecursiveImporter struct {
	IgnorePkgs map[string]bool
	// ignoreされていないパッケージからimportされたパッケージの一覧。
	pkgs map[string]*build.Package
}

// 指定されたパッケージと、それに依存しているパッケージをインポートする。
func (imper *RecursiveImporter) Import(path string, baseDir string) error {
	if imper.pkgs == nil {
		imper.pkgs = map[string]*build.Package{}
	}
	if imper.pkgs[path] != nil {
		// 既にインポート済みなのでスキップする
		return nil
	}
	if imper.IgnorePkgs[path] {
		return nil
	}
	if path == "C" {
		// これはcgo用のパッケージ。
		// 編集可能なgolangのソースコードではないため、無視する。
		return nil
	}

	pkg, err := build.Import(path, baseDir, 0)
	if err != nil {
		return err
	}
	if imper.IgnorePkgs[path] {
		return nil
	}
	log.Println("import", pkg.ImportPath)
	imper.pkgs[pkg.ImportPath] = pkg

	// 依存しているパッケージをインポートする
	for _, imp := range pkg.Imports {
		if err = imper.Import(imp, pkg.Dir); err != nil {
			return err
		}
	}
	return nil
}

// 指定したファイルが依存している全てのパッケージをインポートする。
func (imper *RecursiveImporter) ImportFromFile(gofile string) error {
	abspath, err := filepath.Abs(gofile)
	if err != nil {
		return err
	}
	dirpath := filepath.Dir(abspath)

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, abspath, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	// 依存しているパッケージもインポートする
	for _, imp := range f.Imports {
		if imp.Path.Kind != token.STRING {
			log.Panic("unsupported kind:", imp.Path.Kind)
		}
		impPath := strings.Trim(imp.Path.Value, `"`)

		if err = imper.Import(impPath, dirpath); err != nil {
			return err
		}
	}
	return nil
}

// 指定したパッケージと、そのパッケージが依存している全てのパッケージをインポートする。
func (imper *RecursiveImporter) ImportFromPkg(path string) error {
	return imper.Import(path, "")
}
func (imper RecursiveImporter) Pkgs() map[string]*build.Package {
	return imper.pkgs
}

// copy from github.com/golang/go/src/cmd/go/internal/load/pkg.go
func isStdPkg(path string) bool {
	i := strings.Index(path, "/")
	if i < 0 {
		i = len(path)
	}
	elem := path[:i]
	return !strings.Contains(elem, ".")
}
