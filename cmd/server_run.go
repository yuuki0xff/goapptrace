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
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/yuuki0xff/goapptrace/config"
	"github.com/yuuki0xff/goapptrace/httpserver"
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
	"github.com/yuuki0xff/goapptrace/tracer/protocol"
	"github.com/yuuki0xff/goapptrace/tracer/restapi"
	"github.com/yuuki0xff/goapptrace/tracer/storage"
)

const (
	// クライアントから受信したアイテムのバッファサイズ。
	// 単位はメッセージの個数。
	DefaultReceiveBufferSize = 1 << 16
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

	strg := storage.Storage{
		Root: storage.DirLayout{
			Root: conf.LogsDir(),
		},
	}
	if err := strg.Init(); err != nil {
		fmt.Fprintln(stderr, "ERROR: failed to initialize the storage")
		return err
	}

	if apiAddr == "" {
		apiAddr = config.DefaultApiServerAddr
	}
	if logAddr == "" {
		logAddr = config.DefaultLogServerAddr
	}

	simulatorStore := logutil.StateSimulatorStore{}

	// start API Server
	apiSrv := httpserver.NewHttpServer(apiAddr, restapi.NewRouter(restapi.RouterArgs{
		Config:         conf,
		Storage:        &strg,
		SimulatorStore: &simulatorStore,
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
		Handler: getServerHandler(&strg, &simulatorStore),
		AppName: "TODO", // TODO
		Secret:  "",     // TODO
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
	conf.Servers.ApiServer[1] = &config.ApiServerConfig{
		ServerID: 1,
		Version:  1,
		Addr:     apiSrv.Url(),
	}
	conf.Servers.LogServer[1] = &config.LogServerConfig{
		ServerID: 1,
		Version:  1,
		Addr:     logSrv.ActualAddr(),
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

func getServerHandler(strg *storage.Storage, store *logutil.StateSimulatorStore) protocol.ServerHandler {
	// workerとの通信用。
	// close()されたら、workerは終了するべき。
	chMap := make(map[protocol.ConnID]chan interface{})
	// chanの追加、削除、close()するときはLock()を、chanへの送受信はRLock()をかける。
	var chMapLock sync.RWMutex

	worker := func(ch chan interface{}, id protocol.ConnID) {
		logobj, err := strg.New()
		if err != nil {
			log.Panicf("ERROR: Server: failed to a create Log object: err=%s", err.Error())
		}
		defer func() {
			if err = logobj.Close(); err != nil {
				log.Panicf("failed to close a Log(%s): connID=%d err=%s", logobj.ID, id, err.Error())
			}
			logobj.ReadOnly = true
			if err = logobj.Open(); err != nil {
				log.Panicf("failed to reopen a Log(%s): connID=%d err=%s", logobj.ID, id, err.Error())
			}
		}()

		ss := store.New(logobj.ID)
		defer store.Delete(logobj.ID)
		writeCurrentState := func() {
			for _, fl := range ss.FuncLogs() {
				if err := logobj.AppendFuncLog(fl); err != nil {
					log.Panicln("ERROR: failed to append FuncLog during rotating:", err.Error())
				}
			}
			for _, g := range ss.Goroutines() {
				if err := logobj.AppendGoroutine(g); err != nil {
					log.Panicln("ERROR: failed to append Goroutine during rotating:", err.Error())
				}
			}
			ss.Clear()
		}
		// ログを閉じる前に、現在のStateSimulatorの状態を保存する。
		defer writeCurrentState()
		// このlogobjに対する書き込みを行うのは、worker()のみ。
		// このイベント実行中に他から書き込まれることは考慮しなくてよい。
		logobj.BeforeRotateEventHandler = writeCurrentState

		for rawobj := range ch {
			switch obj := rawobj.(type) {
			case *logutil.RawFuncLog:
				if err := logobj.AppendRawFuncLog(obj); err != nil {
					log.Panicln("failed to append RawFuncLog:", err.Error())
				}
				ss.Next(*obj)
			case *logutil.SymbolsData:
				if err := logobj.AppendSymbolsDiff(obj); err != nil {
					log.Panicln("failed to append Symbols:", err.Error())
				}
			default:
				log.Panicf("unsupported type: %+v", rawobj)
			}
		}
	}

	return protocol.ServerHandler{
		Connected: func(id protocol.ConnID) {
			log.Println("INFO: Server: connected")

			ch := make(chan interface{}, DefaultReceiveBufferSize)
			go worker(ch, id)

			chMapLock.Lock()
			chMap[id] = ch
			chMapLock.Unlock()
		},
		Disconnected: func(id protocol.ConnID) {
			log.Println("INFO: Server: disconnected")

			chMapLock.Lock()
			close(chMap[id])
			delete(chMap, id)
			chMapLock.Unlock()
		},
		Error: func(id protocol.ConnID, err error) {
			log.Printf("ERROR: Server: connID=%d err=%s", id, err.Error())
		},
		Symbols: func(id protocol.ConnID, s *logutil.SymbolsData) {
			chMapLock.RLock()
			chMap[id] <- s
			chMapLock.RUnlock()
		},
		RawFuncLog: func(id protocol.ConnID, f *logutil.RawFuncLog) {
			chMapLock.RLock()
			chMap[id] <- f
			chMapLock.RUnlock()
		},
	}
}
