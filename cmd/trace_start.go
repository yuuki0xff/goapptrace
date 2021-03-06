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
	"github.com/yuuki0xff/goapptrace/tracer/types"
)

// traceStartCmd represents the start command
var traceStartCmd = &cobra.Command{
	Use: "start <log-id> [<name>...]",
	DisableFlagsInUseLine: true,
	Short: "Start tracing of running process",
	Long:  `Start tracing to the specified function. If function name is not given, we traces to all functions.`,
	RunE:  wrap(runTraceStart),
}

func runTraceStart(opt *handlerOpt) error {
	if len(opt.Args) == 0 {
		opt.ErrLog.Println("missing log-id")
		return errInvalidArgs
	}

	logID := opt.Args[0]
	names := opt.Args[1:]

	isRegexp, err := opt.Cmd.Flags().GetBool("regexp")
	if err != nil {
		// 失敗してはいけない
		opt.ErrLog.Panicln(err)
	}

	api, err := opt.Api(context.Background())
	if err != nil {
		opt.ErrLog.Println(err)
		return errApiClient
	}

	tp := tracePattern{
		LogID:    logID,
		Api:      api,
		Names:    names,
		IsRegexp: isRegexp,
		Opt:      opt,
	}
	tp.Matcher()

	sn, err := tp.SymbolNames()
	if err != nil {
		opt.ErrLog.Println(err)
		return errApiClient
	}
	matcher, err := tp.Matcher()
	if err != nil {
		opt.ErrLog.Println(err)
		return errGeneral
	}

	_, err = api.UpdateLogInfo(logID, 10, func(info *types.LogInfo) error {
		for _, s := range sn {
			if matcher(s) {
				info.Metadata.TraceTarget.Funcs = append(info.Metadata.TraceTarget.Funcs, s)
			}
		}
		return nil
	})
	if err != nil {
		opt.ErrLog.Println(err)
		return errGeneral
	}
	return nil
}

func init() {
	traceCmd.AddCommand(traceStartCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// traceStartCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// traceStartCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	traceStartCmd.Flags().BoolP("regexp", "e", false, "Interpret <name> as an regular expression")
}
