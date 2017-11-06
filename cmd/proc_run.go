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
	"os"
	"os/signal"
	"syscall"

	"fmt"

	"log"

	"github.com/spf13/cobra"
	"github.com/yuuki0xff/goapptrace/config"
	"github.com/yuuki0xff/goapptrace/info"
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
	"github.com/yuuki0xff/goapptrace/tracer/protocol"
	"github.com/yuuki0xff/goapptrace/tracer/storage"
)

// procRunCmd represents the run command
var procRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Start processes, and start tracing",
	RunE: wrap(func(conf *config.Config, cmd *cobra.Command, args []string) error {
		conf.WantSave()
		return runProcRun(conf, args)
	}),
}

func runProcRun(conf *config.Config, targets []string) error {
	if len(targets) == 0 {
		targets = conf.Targets.Names()
	}

	strg := storage.Storage{
		Root: storage.DirLayout{
			Root: conf.LogsDir(),
		},
	}
	var logobj *storage.Log
	defer func() {
		if logobj != nil {
			// セッションが異常終了した場合、disconnected eventが発生せずにサーバが終了してしまう。
			// Close()漏れによるファイル破損を防止するため、ここでもClose()しておく
			if err := logobj.Close(); err != nil {
				log.Println(err)
			}
		}
	}()

	if err := strg.Init(); err != nil {
		return err
	}

	addr := fmt.Sprintf("unix:///tmp/goapptrace.%d.sock", os.Getpid())
	srv := protocol.Server{
		Addr: addr,
		Handler: protocol.ServerHandler{
			Connected: func() {
				log.Println("INFO: Server: connected")
				if logobj == nil {
					var err error
					logobj, err = strg.New()
					if err != nil {
						panic(err)
					}
				}
			},
			Disconnected: func() {
				log.Println("INFO: Server: disconnected")
				if err := logobj.Close(); err != nil {
					panic(err)
				}
				logobj = nil
			},
			Error: func(err error) {
				log.Println("ERROR: Server:", err)
			},
			Symbols: func(s *logutil.Symbols) {
				log.Printf("DEBUG: Server: add symbols: %+v\n", s)
				if err := logobj.AppendSymbols(s); err != nil {
					panic(err)
				}
			},
			RawFuncLog: func(f *logutil.RawFuncLogNew) {
				log.Printf("DEBUG: Server: got RawFuncLog: %+v\n", f)
				if err := logobj.AppendFuncLog(f); err != nil {
					panic(err)
				}
			},
		},
		AppName: info.APP_NAME,
		Secret:  "secret", // TODO: set random value
	}
	if err := srv.Listen(); err != nil {
		return err
	}
	go srv.Serve()
	defer srv.Close() // nolint: errcheck

	// set env for child processes
	if err := os.Setenv(info.DEFAULT_LOGSRV_ENV, srv.ActualAddr()); err != nil {
		return err
	}

	for _, targetName := range targets {
		target, err := conf.Targets.Get(config.TargetName(targetName))
		if err != nil {
			return err
		}

		if err := target.Run.Run(); err != nil {
			return err
		}
	}

	var err error
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT)
		<-c
		err = srv.Close()
	}()
	srv.Wait()
	return err
}

func init() {
	procCmd.AddCommand(procRunCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// procRunCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// procRunCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
