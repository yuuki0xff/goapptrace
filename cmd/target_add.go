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
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yuuki0xff/goapptrace/config"
)

// targetAddCmd represents the add command
var targetAddCmd = &cobra.Command{
	Use: "add <name> <path>...",
	DisableFlagsInUseLine: true,
	Short:   "Add to tracing targets",
	Example: targetCmdExample,
	RunE:    wrap(runTargetAdd),
}

func runTargetAdd(opt *handlerOpt) error {
	if len(opt.Args) < 2 {
		opt.ErrLog.Println("Invalid args. See \"target add --help\".")
		return errInvalidArgs
	}

	name := opt.Args[0]
	paths := opt.Args[1:]

	abspaths := make([]string, len(paths))
	for i, p := range paths {
		abspath, err := filepath.Abs(p)
		if err != nil {
			return err
		}
		abspaths[i] = abspath
	}

	opt.Conf.WantSave()
	return opt.Conf.Targets.Add(&config.Target{
		Name:  config.TargetName(name),
		Files: abspaths,
	})
}

func init() {
	targetCmd.AddCommand(targetAddCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// targetAddCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// targetAddCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
