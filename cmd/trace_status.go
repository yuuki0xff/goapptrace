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
	"strconv"

	"github.com/spf13/cobra"
	"github.com/yuuki0xff/goapptrace/config"
)

// traceStatusCmd represents the status command
var traceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of tracer",
	RunE:  wrap(runTraceStatus),
}

func runTraceStatus(opt *handlerOpt) error {
	table := defaultTable(opt.Stdout)
	table.SetHeader([]string{
		"target",
		"files/dirs",
		"has tracing code",
		"tracing",
	})
	if err := opt.Conf.Targets.Walk(nil, func(t *config.Target) error {
		return t.WalkTraces(nil, func(fname string, trace *config.Trace, created bool) error {
			table.Append([]string{
				string(t.Name),
				fname,
				strconv.FormatBool(trace.HasTracingCode),
				strconv.FormatBool(trace.IsTracing),
			})
			return nil
		})
	}); err != nil {
		opt.ErrLog.Println(err)
		return errGeneral
	}
	table.Render()
	return nil
}

func init() {
	traceCmd.AddCommand(traceStatusCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// traceStatusCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// traceStatusCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
