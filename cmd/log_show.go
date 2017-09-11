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

	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
	"github.com/yuuki0xff/goapptrace/config"
	"github.com/yuuki0xff/goapptrace/httpserver"
	"github.com/yuuki0xff/goapptrace/tracer/storage"
)

// logShowCmd represents the show command
var logShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show logs on web browser",
	RunE: wrap(func(conf *config.Config, cmd *cobra.Command, args []string) error {
		notOpenBrowser, err := cmd.Flags().GetBool("no-browser")
		if err != nil {
			return err
		}
		listen, err := cmd.Flags().GetString("listen")
		return runLogShow(conf, args, notOpenBrowser, listen)
	}),
}

func runLogShow(conf *config.Config, targets []string, notOpenBrowser bool, listen string) error {
	strg := &storage.Storage{
		Root: storage.DirLayout{Root: conf.LogsDir()},
	}
	strg.Init()

	srvArgs := &httpserver.ServerArgs{
		Storage: strg,
	}
	srv := httpserver.NewHttpServer(listen, srvArgs)
	if err := srv.Start(); err != nil {
		return err
	}
	fmt.Printf("Started HTTP server on %s\n", srv.Addr())

	if !notOpenBrowser {
		if err := open.Run(fmt.Sprintf("http://%s", srv.Addr())); err != nil {
			return err
		}
	}

	if err := srv.Wait(); err != nil {
		return err
	}
	return nil
}

func init() {
	logCmd.AddCommand(logShowCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// logShowCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// logShowCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	logShowCmd.Flags().BoolP("no-browser", "b", false, "does not open web browser")
	logShowCmd.Flags().StringP("listen", "l", "", "Set listen addr and port (ex: \"127.0.0.1:8080\")")
}
