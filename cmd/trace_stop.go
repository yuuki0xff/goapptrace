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
	"context"

	"github.com/spf13/cobra"
)

// traceStopCmd represents the stop command
var traceStopCmd = &cobra.Command{
	Use: "stop <log-id> [<name>...]",
	DisableFlagsInUseLine: true,
	Short: "Stop tracing of running processes",
	Long:  `Stop tracing to the specified function. If function name is not given, we stops tracing to all functions.`,
	RunE:  wrap(runTraceStop),
}

func runTraceStop(opt *handlerOpt) error {
	if len(opt.Args) == 0 {
		opt.ErrLog.Println("missing tracer-id")
		return errInvalidArgs
	}

	logID := opt.Args[0]
	names := opt.Args[1:]

	api, err := opt.Api(context.Background())
	if err != nil {
		opt.ErrLog.Println(err)
		return errApiClient
	}

	if len(names) == 0 {
		err = api.UpdateTraceTargets(logID, []string{})
		if err != nil {
			opt.ErrLog.Println(err)
			return errGeneral
		}
	} else {
		for _, name := range names {
			err = api.StopTrace(logID, name)
			if err != nil {
				opt.ErrLog.Println(err)
				return errGeneral
			}
		}
	}
	return nil
}

func init() {
	traceCmd.AddCommand(traceStopCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// traceStopCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// traceStopCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
