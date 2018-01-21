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
	"github.com/spf13/cobra"
	"github.com/yuuki0xff/goapptrace/config"
	"github.com/yuuki0xff/goapptrace/tracer/srceditor"
)

// traceOnCmd represents the on command
var traceOnCmd = &cobra.Command{
	Use:   "on",
	Short: "Insert tracing codes into targets",
	RunE: wrap(func(conf *config.Config, cmd *cobra.Command, args []string) error {
		conf.WantSave()
		exportedOnly, err := cmd.Flags().GetBool("exported")
		if err != nil {
			return err
		}
		prefix, err := cmd.Flags().GetString("prefix")
		if err != nil {
			return err
		}

		return runTraceOn(conf, exportedOnly, prefix, args)
	}),
}

func runTraceOn(conf *config.Config, exportedOnly bool, prefix string, targetNames []string) error {
	return conf.Targets.Walk(targetNames, func(t *config.Target) error {
		return t.WalkTraces(t.Files, func(fname string, trace *config.Trace, created bool) error {
			files, err := srceditor.FindFiles(fname)
			if err != nil {
				return err
			}

			editor := &srceditor.CodeEditor{
				ExportedOnly: exportedOnly,
				Prefix:       prefix,
			}
			for _, f := range files {
				if err := editor.Edit(f); err != nil {
					return err
				}
			}

			if created {
				trace.HasTracingCode = true // TODO: currently always true
				trace.IsTracing = true
			} else {
				trace.HasTracingCode = true
			}
			return nil
		})
	})
}

func init() {
	traceCmd.AddCommand(traceOnCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// traceOnCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// traceOnCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	traceOnCmd.Flags().BoolP("exported", "e", false, "Insert tracing code into exorted function only")
	traceOnCmd.Flags().StringP("prefix", "p", "", "Set prefix of import names and variable names")
}
