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

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/yuuki0xff/goapptrace/config"
	"github.com/yuuki0xff/goapptrace/tracer/builder"
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build [-o output] [-i] [packages]",
	Short: "compile packages and dependencies with goapptrace logger",
	Long: `"goapptrace build" is a useful command like "go build".
This command adds logging codes to specified files before build, and build them.
Original source code is not change!
Arguments are compatible with "go build". See "go build --help" to get more information about arguments.`,
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
	log.Println("tmpdir:", tmpdir)
	//defer os.RemoveAll(tmpdir) // nolint: errcheck

	goroot := path.Join(tmpdir, "goroot")
	gopath := path.Join(tmpdir, "gopath")

	b := builder.RepoBuilder{
		Goroot: goroot,
		Gopath: gopath,
		IgnorePkgs: map[string]bool{
			"github.com/yuuki0xff/goapptrace/tracer/logger": true,
		},
		IgnoreStdPkgs: true,
	}
	if err := b.Init(); err != nil {
		fmt.Fprintf(stderr, err.Error()+"\n")
		log.Fatal("Fail")
	}

	isGofiles, err := builder.IsGofiles(targets)
	if err != nil {
		log.Fatal(err)
	}

	// insert logging codes
	if err = b.EditAll(targets); err != nil {
		fmt.Fprintf(stderr, err.Error()+"\n")
		log.Fatal("Fail")
	}
	log.Println("OK")
	//os.Exit(0)

	newTargets := targets
	if isGofiles {
		newTargets = make([]string, len(targets))
		for i := range targets {
			newTargets[i] = path.Join(b.MainPkgDir(), path.Base(targets[i]))
		}
	}

	buildCmd := exec.Command("go", buildArgs(flags, newTargets)...)
	buildCmd.Stdout = stdout
	buildCmd.Stderr = stderr
	buildCmd.Env = buildEnv(goroot, gopath)
	return buildCmd.Run()
}

// "go build"コマンドの実行前にセットするべき環境変数を返す
func buildEnv(goroot, gopath string) []string {
	env := os.Environ()
	env = append(env, "GOROOT="+goroot)
	env = append(env, "GOPATH="+gopath)
	return env
}

// "go build"の引数を返す
func buildArgs(flags *pflag.FlagSet, targets []string) []string {
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
	return append(buildArgs, targets...)
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

	buildCmd.SetFlagErrorFunc(fixFlagName)
}
