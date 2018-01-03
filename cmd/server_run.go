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
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/yuuki0xff/goapptrace/config"
	"github.com/yuuki0xff/goapptrace/httpserver"
	"github.com/yuuki0xff/goapptrace/tracer/protocol"
	"github.com/yuuki0xff/goapptrace/tracer/restapi"
)

// serverRunCmd represents the run command
var serverRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Start log servers",
	RunE: wrap(func(conf *config.Config, cmd *cobra.Command, args []string) error {
		apiAddr, _ := cmd.Flags().GetString("listen-api")
		logAddr, _ := cmd.Flags().GetString("listen-log")
		return runServerRun(
			conf, cmd.OutOrStdout(), cmd.OutOrStderr(),
			apiAddr, logAddr,
		)
	}),
}

func runServerRun(conf *config.Config, stdout io.Writer, stderr io.Writer, apiAddr, logAddr string) error {
	if len(conf.Servers.ApiServer) > 0 {
		// API server SHOULD one instance.
		fmt.Fprintln(stderr, "ERROR: API server is already running")
		return nil
	}
	if len(conf.Servers.LogServer) > 0 {
		// Log server SHOULD one instance.
		fmt.Fprintln(stderr, "ERROR: Log server is already running")
		return nil
	}

	if apiAddr == "" {
		apiAddr = config.DefaultApiServerAddr
	}
	if logAddr == "" {
		logAddr = config.DefaultLogServerAddr
	}

	// start API Server
	apiSrv := httpserver.NewHttpServer(apiAddr, restapi.NewRouter(restapi.RouterArgs{
		Config:  conf,
		Storage: nil, // TODO
	}))
	if err := apiSrv.Start(); err != nil {
		fmt.Fprintln(stderr, "ERROR: failed to start the API server:", err)
		return err
	}
	defer func() {
		apiSrv.Stop()
		if err := apiSrv.Wait(); err != nil {
			fmt.Fprintln(stderr, "ERROR: failed to stop the API server:", err)
		}
	}()

	// start Log Server
	logSrv := protocol.Server{
		Addr:    "tcp://" + logAddr,
		Handler: protocol.ServerHandler{},
		AppName: "TODO",
		Secret:  "",
	}
	if err := logSrv.Listen(); err != nil {
		fmt.Fprintln(stderr, "ERROR: failed to start the Log server:", err)
		return err
	}
	go logSrv.Serve()
	defer func() {
		defer logSrv.Wait()
		if err := logSrv.Close(); err != nil {
			fmt.Fprintln(stderr, "ERROR: failed to stop the Log server:", err)
		}
	}()

	// add servers to config, and save
	conf.Servers.ApiServer[1] = &config.ApiServerConfg{
		Addr: apiSrv.Addr(),
	}
	conf.Servers.LogServer[1] = &config.LogServerConfig{
		Addr: logSrv.ActualAddr(),
	}
	conf.WantSave()
	if err := conf.Save(); err != nil {
		fmt.Fprintln(stderr, "ERROR: cannot write to the config file:", err)
	}

	// wait until a signal is received
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)
	<-sigCh

	// remove servers from config
	conf.Servers = *config.NewServers()
	conf.WantSave()
	if err := conf.Save(); err != nil {
		fmt.Fprintln(stderr, "ERROR: cannot write to the config file:", err)
	}
	return nil
}

func init() {
	serverCmd.AddCommand(serverRunCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serverRunCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// serverRunCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	serverRunCmd.Flags().StringP("listen-api", "p", "", "Address and port for REST API Server")
	serverRunCmd.Flags().StringP("listen-log", "P", "", "Address and port for Log Server")
}
