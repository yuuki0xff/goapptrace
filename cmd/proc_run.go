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

	"log"

	"sync"

	"fmt"

	"time"

	"github.com/spf13/cobra"
	"github.com/yuuki0xff/goapptrace/config"
	"github.com/yuuki0xff/goapptrace/info"
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
	"github.com/yuuki0xff/goapptrace/tracer/protocol"
	"github.com/yuuki0xff/goapptrace/tracer/storage"
)

const (
	CloseDelayDuration = 3 * time.Second
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

	// key: protocol.ConnID
	// value: *storage.LogWriter
	var logobjs sync.Map
	defer logobjs.Range(func(key, value interface{}) bool {
		id := key.(protocol.ConnID)
		logobjs.Delete(key)
		logobj := value.(*storage.LogWriter)
		// セッションが異常終了した場合、disconnected eventが発生せずにサーバが終了してしまう。
		// Close()漏れによるファイル破損を防止するため、ここでもClose()しておく
		if err := logobj.Close(); err != nil {
			log.Printf("failed to close LogWriter(%s) file: %s", id, err.Error())
		}
		return true
	})
	getLog := func(id protocol.ConnID) *storage.LogWriter {
		value, ok := logobjs.Load(id)
		if ok == false {
			log.Panicf("ERROR: Server: ConnID(%s) not found", id)
		}
		l := value.(*storage.Log)
		w, err := l.Writer()
		if err != nil {
			log.Panicf("cast error: %s", err.Error())
		}
		return w
	}

	if err := strg.Init(); err != nil {
		return err
	}

	// use ephemeral port for communication with child process
	srv := protocol.Server{
		Addr: "",
		Handler: protocol.ServerHandler{
			Connected: func(id protocol.ConnID) {
				log.Println("INFO: Server: connected")

				// create a LogWriter object
				logobj, err := strg.New()
				if err != nil {
					log.Panicf("ERROR: Server: failed to a create LogWriter object: err=%s", err.Error())
				}
				if _, loaded := logobjs.LoadOrStore(id, logobj); loaded {
					log.Panicf("ERROR: Server: failed to a store LogWriter object. this process MUST success")
				}
			},
			Disconnected: func(id protocol.ConnID) {
				log.Println("INFO: Server: disconnected")

				logobj := getLog(id)
				logobjs.Delete(id)
				if err := logobj.Close(); err != nil {
					log.Panicf("ERROR: Server: failed to close a LogWriter object: err=%s", err.Error())
				}
			},
			Error: func(id protocol.ConnID, err error) {
				// TODO: check ConnID
				log.Println("ERROR: Server:", err)
			},
			Symbols: func(id protocol.ConnID, s *logutil.Symbols) {
				log.Printf("DEBUG: Server: add symbols: %+v\n", s)

				logobj := getLog(id)
				if err := logobj.AppendSymbols(s); err != nil {
					panic(err)
				}
			},
			RawFuncLog: func(id protocol.ConnID, f *logutil.RawFuncLogNew) {
				log.Printf("DEBUG: Server: got RawFuncLog: %+v\n", f)
				logobj := getLog(id)
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

	// set env for child processes
	if err := os.Setenv(info.DEFAULT_LOGSRV_ENV, srv.ActualAddr()); err != nil {
		return err
	}

	var lastErr error
	wg := sync.WaitGroup{}
	for _, targetName := range targets {
		target, err := conf.Targets.Get(config.TargetName(targetName))
		if err != nil {
			return err
		}
		proc, err := target.Run.Start()
		if err != nil {
			return err
		}
		wg.Add(1)
		go func() {
			if err := proc.Wait(); err != nil {
				wrapped := fmt.Errorf("failed run a command (%s): %s", proc.Args, err.Error())
				lastErr = wrapped
				log.Printf("WARN: %s", wrapped)
			}
			wg.Done()
		}()
	}

	go func() {
		// close server when all child process was exited
		wg.Wait()
		log.Println("DEBUG: all child process was exited. the server is going exit")

		// wait for receive all packets that staying in the network
		time.Sleep(CloseDelayDuration)

		if err := srv.Close(); err != nil {
			lastErr = err
		}
	}()
	go func() {
		// close server when a signal was received
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT)
		<-c
		log.Println("DEBUG: signal was received. the server is going exit")
		if err := srv.Close(); err != nil {
			lastErr = err
		}
	}()
	srv.Wait()
	return lastErr
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
