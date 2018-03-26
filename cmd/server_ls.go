// Copyright © 2017 yuuki0xff
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
)

// serverLsCmd represents the ls command
var serverLsCmd = &cobra.Command{
	Use: "ls",
	DisableFlagsInUseLine: true,
	Short: "Show log servers",
	RunE:  wrap(runServerLs),
}

func runServerLs(opt *handlerOpt) error {
	stdout := opt.Stdout

	fmt.Fprintln(stdout, "API Servers")
	fmt.Fprintln(stdout, "================")
	if len(opt.Conf.Servers.ApiServer) > 0 {
		apiTbl := defaultTable(stdout)
		apiTbl.SetHeader([]string{"ID", "Address"})
		for id, s := range opt.Conf.Servers.ApiServer {
			apiTbl.Append([]string{
				fmt.Sprint(id),
				s.Addr,
			})
		}
		apiTbl.Render()
	} else {
		fmt.Fprintln(stdout, "API Server is not running")
	}

	fmt.Fprintln(stdout)

	fmt.Fprintln(stdout, "Log Servers")
	fmt.Fprintln(stdout, "================")
	if len(opt.Conf.Servers.LogServer) > 0 {
		logTbl := defaultTable(stdout)
		logTbl.SetHeader([]string{"ID", "Address"})
		for id, s := range opt.Conf.Servers.LogServer {
			logTbl.Append([]string{
				fmt.Sprint(id),
				s.Addr,
			})
		}
		logTbl.Render()
	} else {
		fmt.Fprintln(stdout, "Log Server is not running")
	}
	return nil
}

func init() {
	serverCmd.AddCommand(serverLsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serverLsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// serverLsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
