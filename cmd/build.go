// Copyright © 2018 yuuki0xff <yuuki0xff@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"fmt"
	"go/importer"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/yuuki0xff/goapptrace/config"
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Add trace codes and compile the packages",
	Long: `Build is an useful command like "go build".
Add trace codes to specified files before build, and build them.
Source code is no change!`,
	RunE: wrap(func(conf *config.Config, cmd *cobra.Command, args []string) error {
		fmt.Println("build called")
		return runBuild(conf, cmd.Flags(), cmd.OutOrStdout(), cmd.OutOrStderr(), args)
	}),
}

func runBuild(conf *config.Config, flags *pflag.FlagSet, stdout, stderr io.Writer, targets []string) error {
	tmpdir, err := ioutil.TempDir("", ".goapptrace.build")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpdir) // nolint: errcheck

	goroot := path.Join(tmpdir, "goroot")
	gopath := path.Join(tmpdir, "gopath")

	// TODO: insert trace code
	ok, err := isGofiles(targets)
	if err != nil {
		fmt.Fprintf(stderr, err.Error())
		return err
	}
	if ok {
		log.Println("importing")
		err = parseGofiles(targets)
		if err != nil {
			log.Panic(err)
		}
		os.Exit(0)
	} else {
		// packagesとして見なす
		// TODO:
	}
	log.Panic("FAIL")

	buildCmd := exec.Command("go", buildArgs(flags)...)
	buildCmd.Stdout = stdout
	buildCmd.Stderr = stderr
	buildCmd.Env = buildEnv(goroot, gopath)
	return buildCmd.Run()
}

// 全てのファイルが".go"で終わるファイルなら、trueを返す
func isGofiles(files []string) (bool, error) {
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

func parseGofiles(gofiles []string) error {
	// インポートされたパッケージの一覧
	imports := map[string]bool{}

	imper := importer.For("source", nil)
	var importPackage func(path string) error
	importPackage = func(path string) error {
		if imports[path] {
			// 既にインポート済みなのでスキップする
			return nil
		}
		if path == "C" {
			// cgo用のimportなので、無視する。
			return nil
		}
		log.Println("import", path)

		pkg, err := imper.Import(path)
		if err != nil {
			return err
		}

		// 依存しているパッケージをインポートする
		for _, imp := range pkg.Imports() {
			if err = importPackage(imp.Path()); err != nil {
				return err
			}
		}
		return nil
	}

	fset := token.NewFileSet()
	for _, fname := range gofiles {
		f, err := parser.ParseFile(fset, fname, nil, parser.ParseComments)
		if err != nil {
			return err
		}

		// 依存しているパッケージもインポートする
		for _, imp := range f.Imports {
			if imp.Path.Kind != token.STRING {
				log.Panic("unsupported kind:", imp.Path.Kind)
			}
			impPath := strings.Trim(imp.Path.Value, `"`)

			if err = importPackage(impPath); err != nil {
				return err
			}
		}
	}

	//ast.NewPackage(fset,)
	// TODO
	return nil
}

//func isPackages(pkgs []string) bool {
//	for _, p := range pkgs {
//		gopath := os.Getenv("GOPATH")
//		goroot := runtime.GOROOT()
//
//		if isDir(p) || isDir(path.Join(gopath, p)) || isDir(path.Join(goroot, p)) {
//			continue
//		}
//		return false
//	}
//	return true
//}
//
//func isDir(path string) bool {
//	finfo, err := os.Stat(path)
//	if err != nil {
//		if !os.IsNotExist(err) {
//			log.Panic(errors.Wrap(err,"not found package"))
//		}
//		// 指定されたpathは存在しない
//		return false
//	}
//	return finfo.IsDir()
//}

// "go build"コマンドの実行前にセットするべき環境変数を返す
func buildEnv(goroot, gopath string) []string {
	env := os.Environ()
	env = append(env, "GOROOT="+goroot)
	env = append(env, "GOPATH="+gopath)
	return env
}

// "go build"の引数を返す
func buildArgs(flags *pflag.FlagSet) []string {
	buildArgs := []string{"build"}
	flags.Visit(func(flag *pflag.Flag) {
		var flagname string
		if flag.Shorthand == "" {
			flagname = "-" + flag.Shorthand
		} else {
			flagname = "-" + flag.Name
		}

		value := flag.Value.String()
		switch flag.Value.Type() {
		case "bool":
			if value == "true" {
				buildArgs = append(buildArgs, flagname)
			}
		case "string":
			if value != "" {
				buildArgs = append(buildArgs, flagname, value)
			}
		case "int":
			if value != "0" {
				buildArgs = append(buildArgs, flagname, value)
			}
		default:
			log.Panicf("invalid type name: %s", flag.Value.Type())
		}
	})
	return buildArgs
}

func init() {
	RootCmd.AddCommand(buildCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// buildCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// buildCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	// "go build" flags
	// TODO: "go build"の引数との互換性を改善する。
	//       現時点では、引数名の前に"--"がつくため、厳密には"go build"の引数とは互換性がない。
	//       引数の解析エラーで発生するFlagErrorFuncで、フラグ名を修正して自分自身を呼び出すとよい。
	buildCmd.Flags().StringP("o", "o", "", "forces build to write the resulting executable or object to the named output file.")
	buildCmd.Flags().BoolP("i", "i", false, "install the packages that are dependencies of the target.")
	buildCmd.Flags().BoolP("a", "a", false, "force rebuilding of packages that are already up-to-date.")
	buildCmd.Flags().BoolP("n", "n", false, "print the commands but do not run them.")
	buildCmd.Flags().IntP("p", "p", 0, "specifies the number of threads/commands to run.")
	buildCmd.Flags().BoolP("race", "", false, "enable data race detection.")
	buildCmd.Flags().BoolP("msan", "", false, "enable interoperation with memory sanitizer.")
	buildCmd.Flags().BoolP("v", "v", false, "print the names of packages as they are compiled.")
	buildCmd.Flags().BoolP("work", "", false, "print the name of the temporary work directory and do not delete it when exiting.")
	buildCmd.Flags().BoolP("x", "x", false, "print the commands.")
	buildCmd.Flags().StringP("asmflags", "", "", "arguments to pass on each go tool asm invocation.")
	buildCmd.Flags().StringP("buildmode", "", "", "build mode to use. See 'go help buildmode' for more.")
	buildCmd.Flags().StringP("compiler", "", "", "name of compiler to use, as in runtime.Compiler (gccgo or gc).")
	buildCmd.Flags().StringP("gccgoflags", "", "", "arguments to pass on each gccgo compiler/linker invocation.")
	buildCmd.Flags().StringP("gcflags", "", "", "arguments to pass on each go tool compile invocation.")
	buildCmd.Flags().StringP("installsuffix", "", "", "a suffix to use in the name of the package installation directory.")
	buildCmd.Flags().StringP("ldflags", "", "", "arguments to pass on each go tool link invocation.")
	buildCmd.Flags().BoolP("linkshared", "", false, "link against shared libraries previously created with -buildmode=shared.")
	buildCmd.Flags().StringP("pkgdir", "", "", "install and load all packages from dir instead of the usual locations.")
	buildCmd.Flags().StringP("tags", "", "", "a space-separated list of build tags to consider satisfied during the build.")
	buildCmd.Flags().StringP("toolexec", "", "", "a program to use to invoke toolchain programs like vet and asm.")

	// golang標準のflagパッケージの形式の引数に対応するため、
	// 引数名のprefixを必要に応じて"-"から"--"に変換する。
	buildCmd.SetFlagErrorFunc(func(command *cobra.Command, e error) error {
		// 定義済みの長いフラグ名 ("--"は含まない)
		flagNames := map[string]bool{}
		buildCmd.Flags().VisitAll(func(flag *pflag.Flag) {
			flagNames[flag.Name] = true
		})

		var converted bool
		args := []string{}
		// TODO: fix flag names
		for _, arg := range os.Args[1:] {
			if strings.HasPrefix(arg, "-") && flagNames[arg[1:]] {
				// "-flag"から"--flag"形式に変換する。
				args = append(args, "--"+arg[1:])
				converted = true
			} else {
				args = append(args, arg)
			}
		}

		if !converted {
			// 引数の変換が行えないにも関わらずエラーが発生した状況である。
			// 間違った引数を与えていた可能性があるので、ここで実行を中断。
			return e
		}

		exe, err := os.Executable()
		if err != nil {
			return err
		}

		cmd := exec.Command(exe, args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	})
}
