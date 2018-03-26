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
	"github.com/spf13/cobra"
	"github.com/yuuki0xff/goapptrace/logviewer"
)

// tuiCmd represents the tui command
var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "launch LogViewer client",
	RunE:  wrap(runTuiCmd),
}

func runTuiCmd(opt *handlerOpt) error {
	var logID string
	targets := opt.Args

	if len(targets) >= 2 {
		opt.ErrLog.Println("Too many arguments. length of args should be less than 2")
		return errInvalidArgs
	} else if len(targets) == 1 {
		logID = targets[0]
	}

	api, err := opt.Api(nil)
	if err != nil {
		opt.ErrLog.Println(err)
		return errGeneral
	}

	t := logviewer.UICoordinator{
		Config: opt.Conf,
		Api:    &api.Client,
		LogID:  logID,
	}
	err = t.Run()
	if err != nil {
		opt.ErrLog.Println(err)
		return errGeneral
	}
	return nil
}

func init() {
	RootCmd.AddCommand(tuiCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// tuiCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// tuiCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
