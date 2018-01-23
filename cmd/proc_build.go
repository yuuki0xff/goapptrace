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
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yuuki0xff/goapptrace/config"
)

// procBuildCmd represents the build command
var procBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build with tracing codes",
	RunE: wrap(func(conf *config.Config, cmd *cobra.Command, args []string) error {
		conf.WantSave()
		return runProcBuild(conf, args)
	}),
}

func runProcBuild(conf *config.Config, targets []string) error {
	// TODO: "build"コマンドとの違いを説明する
	if len(targets) == 0 {
		targets = conf.Targets.Names()
	}

	for _, targetName := range targets {
		target, err := conf.Targets.Get(config.TargetName(targetName))
		if err != nil {
			return err
		}

		buildProc, err := target.Build.Run()
		if err != nil {
			return fmt.Errorf("failed to run a command (%s): %s", buildProc.Args, err.Error())
		}
	}
	return nil
}

func init() {
	procCmd.AddCommand(procBuildCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// procBuildCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// procBuildCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
