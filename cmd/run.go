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
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/yuuki0xff/goapptrace/info"
	"github.com/yuuki0xff/goapptrace/tracer/restapi"
)

var runFlags = mergeFlagNames(sharedFlagNames(), map[string]bool{
	"exec": true,
})

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use: "run [build flags] [-exec xprog] -- <gofiles>...",
	DisableFlagsInUseLine: true,
	Short: "compile and run Go program",
	Long: `"goapptrace run" is a useful command like "go run".
This command compiles specified files with logging codes, and execute them.
Arguments are compatible with "go run". See "go run --help" to get more information about arguments.`,
	RunE: wrap(runRun),
}

func runRun(opt *handlerOpt) error {
	srv, err := opt.LogServer()
	if err != nil {
		opt.ErrLog.Println(err)
		return errGeneral
	}

	targets := opt.Args
	files, cmdArgs := separateGofilesAndArgs(targets)
	if len(files) == 0 {
		opt.ErrLog.Println("No go files listed")
		return errGeneral
	}

	tmpdir, err := ioutil.TempDir("", ".goapptrace.run")
	if err != nil {
		opt.ErrLog.Println(err)
		return errGeneral
	}
	log.Println("tmpdir:", tmpdir)
	//defer os.RemoveAll(tmpdir) // nolint: errcheck

	b, err := prepareRepo(tmpdir, files, opt.Conf)
	if err != nil {
		opt.ErrLog.Println(err)
		return errGeneral
	}

	// ビルド対象のファイルパスを修正する。
	newFiles := make([]string, len(files))
	for i := range files {
		dir, err := b.MainPkgDir(files[i])
		if err != nil {
			opt.ErrLog.Println(err)
			return errGeneral
		}
		newFiles[i] = path.Join(dir, path.Base(files[i]))
	}

	// ignore an error of "Subprocess launching with variable" because arguments are specified by the trusted user.
	runCmd := exec.Command("go", runArgs(opt.Cmd.Flags(), newFiles, cmdArgs)...) // nolint: gas
	runCmd.Stdin = opt.Stdin
	runCmd.Stdout = opt.Stdout
	runCmd.Stderr = opt.Stderr
	// 実行用の環境変数を追加しなきゃ鳴らない
	runCmd.Env = append(os.Environ(), runEnv(srv, b.Goroot, b.Gopath, files)...)
	return runCmd.Run()
}

// "go run"の引数を返す
func runArgs(flagset *pflag.FlagSet, files, cmdArgs []string) []string {
	return append(append(append(
		[]string{"run"},
		toShortPrefixFlag(flagset, runFlags)...),
		files...),
		cmdArgs...)
}

// "go run"コマンドの実行前にセットするべき環境変数を返す
func runEnv(srv restapi.ServerStatus, goroot, gopath string, files []string) []string {
	env := buildEnv(goroot, gopath, files)
	env = append(env, info.DEFAULT_LOGSRV_ENV+"="+srv.Addr)
	return env
}

func separateGofilesAndArgs(args []string) (files, cmdArgs []string) {
	i := 0
	for i < len(args) && strings.HasSuffix(args[i], ".go") {
		i++
	}
	files, cmdArgs = args[:i], args[i:]
	return
}

func init() {
	RootCmd.AddCommand(runCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// runCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// runCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	runCmd.Flags().StringP("exec", "", "", "invoke the binary using specified command")
	runCmd.Flags().AddFlagSet(sharedFlags())

	runCmd.SetFlagErrorFunc(fixFlagName(runFlags))
}
