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
	"io"

	"github.com/spf13/cobra"
)

// logQueryCmd represents the query command
var logQueryCmd = &cobra.Command{
	Use: "query <id> <SQL>",
	DisableFlagsInUseLine: true,
	Short: "Execute a SELECT query",
	RunE:  wrap(runLogQuery),
}

func runLogQuery(opt *handlerOpt) error {
	if len(opt.Args) < 1 {
		opt.ErrLog.Println("Log ID and SQL statement are not specified.")
		return errInvalidArgs
	} else if len(opt.Args) < 2 {
		opt.ErrLog.Println("SQL statement is not specified.")
		return errInvalidArgs
	} else if len(opt.Args) > 2 {
		opt.ErrLog.Println("Multiple SQL queries cannot be specified.")
		return errInvalidArgs
	}

	api, err := opt.Api(nil)
	if err != nil {
		opt.ErrLog.Println(err)
		return errGeneral
	}

	id, query := opt.Args[0], opt.Args[1]
	r, err := api.SearchRaw(id, query)
	if err != nil {
		opt.ErrLog.Println(err)
		return errGeneral
	}
	defer r.Close() // nolint
	_, err = io.Copy(opt.Stdout, r)
	if err != nil {
		opt.ErrLog.Println(err)
		return errIo
	}
	return nil
}

func init() {
	logCmd.AddCommand(logQueryCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// logQueryCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// logQueryCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
