// Copyright © 2017 yuuki0xff <yuuki0xff@gmail.com>
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
	"strings"

	"github.com/spf13/cobra"
	"github.com/yuuki0xff/goapptrace/config"
)

// targetSetBuildCmd represents the setBuild command
var targetSetBuildCmd = &cobra.Command{
	Use: "set-build <name> <cmd>...",
	DisableFlagsInUseLine: true,
	Short:   "Set the custom build processes instead of 'go build'",
	Example: targetCmdExample,
	RunE:    wrap(runTargetSetBuild),
}

func runTargetSetBuild(opt *handlerOpt) error {
	if len(opt.Args) < 1 {
		opt.ErrLog.Println("Invalid args. See \"target set-build --help\".")
		return errInvalidArgs
	}

	name := opt.Args[0]
	cmds := opt.Args[1:]
	useShell, err := opt.Cmd.Flags().GetBool("shell")
	if err != nil {
		opt.ErrLog.Println(err)
		return errInvalidArgs
	}

	t, err := opt.Conf.Targets.Get(config.TargetName(name))
	if err != nil {
		opt.ErrLog.Println(err)
		return errGeneral
	}

	if useShell {
		t.Build.Args = []string{"/bin/bash", "-c", strings.Join(cmds, " ")}
	} else {
		t.Build.Args = cmds
	}
	opt.Conf.WantSave()
	return nil
}

func init() {
	targetCmd.AddCommand(targetSetBuildCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// targetSetBuildCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// targetSetBuildCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	targetSetBuildCmd.Flags().BoolP("shell", "s", false, "Execute throught shell")
}
