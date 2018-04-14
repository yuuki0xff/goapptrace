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
	"bytes"
	"context"
	"io"
	"strconv"

	"github.com/deckarep/golang-set"
	"github.com/spf13/cobra"
	"github.com/yuuki0xff/goapptrace/tracer/types"
)

// traceLsCmd represents the traceLs command
var traceLsCmd = &cobra.Command{
	Use: "ls <log-id>",
	DisableFlagsInUseLine: true,
	Short: "List all functions",
	RunE:  wrap(runTraceLs),
}

func runTraceLs(opt *handlerOpt) error {
	if len(opt.Args) == 0 {
		opt.ErrLog.Println("missing tracer-id")
		return errInvalidArgs
	}
	logID := opt.Args[0]

	api, err := opt.Api(context.Background())
	if err != nil {
		opt.ErrLog.Println(err)
		return errApiClient
	}
	sym, err := api.Symbols(logID)
	if err != nil {
		opt.ErrLog.Println(err)
		return errGeneral
	}
	info, err := api.LogInfo(logID)
	if err != nil {
		opt.ErrLog.Println(err)
		return errGeneral
	}

	enabledFuncs := mapset.NewSet()
	for _, f := range info.Metadata.TraceTarget.Funcs {
		enabledFuncs.Add(f)
	}

	var buf bytes.Buffer
	tbl := defaultTable(&buf)
	tbl.SetHeader([]string{
		"name",
		"status",
	})
	err = sym.Save(func(data types.SymbolsData) error {
		for _, f := range data.Funcs {
			tbl.Append([]string{
				f.Name,
				strconv.FormatBool(enabledFuncs.Contains(f.Name)),
			})
		}
		return nil
	})
	if err != nil {
		opt.ErrLog.Println(err)
		return errGeneral
	}
	tbl.Render()

	_, err = io.Copy(opt.Stdout, &buf)
	if err != nil {
		return errIo
	}
	return nil
}

func init() {
	traceCmd.AddCommand(traceLsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// traceLsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// traceLsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
