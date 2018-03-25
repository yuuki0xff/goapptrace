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
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/yuuki0xff/goapptrace/config"
)

// logQueryCmd represents the query command
var logQueryCmd = &cobra.Command{
	Use: "query <id> <SQL>",
	DisableFlagsInUseLine: true,
	Short: "Execute a SELECT query",
	RunE: wrap(func(conf *config.Config, cmd *cobra.Command, args []string) error {
		stderr := cmd.OutOrStderr()
		if len(args) < 2 {
			fmt.Fprintln(stderr, "ERROR: SQL statement is not specified.")
			return errors.New("invalid args")
		} else if len(args) > 2 {
			fmt.Fprintln(stderr, "ERROR: Multiple SQL queries cannot be specified.")
			return errors.New("invalid args")
		}
		id, query := args[0], args[1]
		if err := runLogQuery(conf, cmd, id, query); err != nil {
			fmt.Fprintln(stderr, "ERROR:", err.Error())
			return err
		}
		return nil
	}),
}

func runLogQuery(conf *config.Config, cmd *cobra.Command, id, query string) error {
	stdout := cmd.OutOrStdout()

	apiNoctx, err := getAPIClient(conf)
	if err != nil {
		return err
	}
	api := apiNoctx.WithCtx(context.Background())

	r, err := api.SearchRaw(id, query)
	if err != nil {
		return err
	}
	defer r.Close() // nolint
	_, err = io.Copy(stdout, r)
	return err
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
