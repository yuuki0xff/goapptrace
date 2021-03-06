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
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/yuuki0xff/goapptrace/info"
	"github.com/yuuki0xff/goapptrace/tracer/builder"
)

var buildFlags = mergeFlagNames(sharedFlagNames(), map[string]bool{
	"o": true,
	"i": true,
})

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use: "build [-o output] [-i] <package>|<file>...",
	DisableFlagsInUseLine: true,
	Short: "compile packages and dependencies with goapptrace logger",
	Long: `"goapptrace build" is a useful command like "go build".
This command adds logging codes to specified files before build, and build them.
Original source code is not change!
Arguments are compatible with "go build". See "go build --help" to get more information about arguments.`,
	RunE: wrap(runBuild),
}

func runBuild(opt *handlerOpt) error {
	tmpdir, err := ioutil.TempDir("", ".goapptrace.build")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpdir) // nolint: errcheck

	targets := opt.Args
	if len(targets) == 0 {
		opt.ErrLog.Println("no packages or files given.")
		return errInvalidArgs
	}
	b, err := prepareRepo(tmpdir, targets, opt.Conf)
	if err != nil {
		opt.ErrLog.Println(err)
		return errGeneral
	}

	var newTargets []string
	isGofiles, err := builder.IsGofiles(targets)
	if err != nil {
		opt.ErrLog.Println(err)
		return errGeneral
	}
	if isGofiles {
		// ビルド対象のファイルパスを修正する。
		newTargets = make([]string, len(targets))
		for i := range targets {
			dir, err := b.MainPkgDir(targets[i])
			if err != nil {
				return err
			}
			newTargets[i] = path.Join(dir, path.Base(targets[i]))
		}
	} else {
		// import pathは変更不要。
		newTargets = targets
	}

	// ignore an error of "Subprocess launching with variable" because arguments are specified by the trusted user.
	buildCmd := exec.Command("go", buildArgs(opt.Cmd.Flags(), newTargets)...) // nolint: gas
	buildCmd.Stdout = opt.Stdout
	buildCmd.Stderr = opt.Stderr
	buildCmd.Env = append(os.Environ(), buildEnv(b.Goroot, b.Gopath, newTargets)...)
	return buildCmd.Run()
}

// "go build"コマンドの実行前にセットするべき環境変数を返す。
// filesはmainパッケージのパス、またはmainパッケージのファイルのリスト。
func buildEnv(goroot, gopath string, files []string) (env []string) {
	if len(files) == 0 {
		panic("files MUST NOT empty")
	}
	f := files[0]
	var appName string
	if strings.HasSuffix(f, ".go") {
		idx := strings.LastIndex(f, "/")
		if idx >= 0 {
			appName = f[idx+1:]
		} else {
			appName = f
		}
		appName = strings.TrimSuffix(appName, ".go")
	} else {
		abspath, err := filepath.Abs(f)
		if err != nil {
			abspath = f
		}
		abspath2 := strings.TrimRight(abspath, "/")
		arr := strings.Split(abspath2, "/")
		appName = arr[len(arr)-1]
	}

	env = append(env, info.DefaultAppNameEnv+"="+appName)
	env = append(env, "GOROOT="+goroot)
	env = append(env, "GOPATH="+gopath)
	return env
}

// "go build"の引数を返す
func buildArgs(flagset *pflag.FlagSet, targets []string) []string {
	return append(append(
		[]string{"build"},
		toShortPrefixFlag(flagset, buildFlags)...),
		targets...)
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
	buildCmd.Flags().StringP("o", "o", "", "forces build to write the resulting executable or object to the named output file.")
	buildCmd.Flags().BoolP("i", "i", false, "install the packages that are dependencies of the target.")
	buildCmd.Flags().AddFlagSet(sharedFlags())

	buildCmd.SetFlagErrorFunc(fixFlagName(buildFlags))
}
