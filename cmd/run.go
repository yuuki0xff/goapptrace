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
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/yuuki0xff/goapptrace/config"
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
	RunE: wrap(func(conf *config.Config, cmd *cobra.Command, args []string) error {
		return runRun(conf, cmd.Flags(), os.Stdin, cmd.OutOrStdout(), cmd.OutOrStderr(), args)
	}),
}

func runRun(conf *config.Config, flags *pflag.FlagSet, stdin io.Reader, stdout, stderr io.Writer, targets []string) error {
	//cmd := exec.Command("echo", buildArgs(flags, targets)...)
	//cmd.Stdout = stdout
	//cmd.Run()
	//os.Exit(0)

	srv, err := getLogServer(conf)
	if err != nil {
		log.Panic(err)
	}

	files, cmdArgs := separateGofilesAndArgs(targets)
	if len(files) == 0 {
		log.Panic("goapptrace run: no go files listed")
	}

	tmpdir, err := ioutil.TempDir("", ".goapptrace.run")
	if err != nil {
		return err
	}
	log.Println("tmpdir:", tmpdir)
	//defer os.RemoveAll(tmpdir) // nolint: errcheck

	b, err := prepareRepo(tmpdir, files, conf)
	if err != nil {
		fmt.Fprintf(stderr, err.Error()+"\n")
		log.Fatal("Fail")
	}

	// ビルド対象のファイルパスを修正する。
	newFiles := make([]string, len(files))
	for i := range files {
		dir, err := b.MainPkgDir(files[i])
		if err != nil {
			return err
		}
		newFiles[i] = path.Join(dir, path.Base(files[i]))
	}

	// ignore an error of "Subprocess launching with variable" because arguments are specified by the trusted user.
	runCmd := exec.Command("go", runArgs(flags, newFiles, cmdArgs)...) // nolint: gas
	runCmd.Stdin = stdin
	runCmd.Stdout = stdout
	runCmd.Stderr = stderr
	// 実行用の環境変数を追加しなきゃ鳴らない
	runCmd.Env = append(os.Environ(), runEnv(srv, b.Goroot, b.Gopath)...)
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
func runEnv(srv restapi.ServerStatus, goroot, gopath string) []string {
	env := buildEnv(goroot, gopath)
	env = append(env, procRunEnv(srv)...)
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
