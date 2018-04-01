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
	"log"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/yuuki0xff/goapptrace/config"
	"github.com/yuuki0xff/goapptrace/httpserver"
	"github.com/yuuki0xff/goapptrace/tracer/protocol"
	"github.com/yuuki0xff/goapptrace/tracer/restapi"
	"github.com/yuuki0xff/goapptrace/tracer/simulator"
	"github.com/yuuki0xff/goapptrace/tracer/storage"
	"github.com/yuuki0xff/goapptrace/tracer/types"
)

const (
	// クライアントから受信したアイテムのバッファサイズ。
	// 単位はメッセージの個数。
	DefaultReceiveBufferSize = 128
)

// serverRunCmd represents the run command
var serverRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Start log servers",
	RunE:  wrap(runServerRun),
}

func runServerRun(opt *handlerOpt) error {
	apiAddr, _ := opt.Cmd.Flags().GetString("listen-api")
	logAddr, _ := opt.Cmd.Flags().GetString("listen-log")

	if len(opt.Conf.Servers.ApiServer) > 0 {
		// API server SHOULD one instance.
		opt.ErrLog.Println("API server is already running")
		return errGeneral
	}
	if len(opt.Conf.Servers.LogServer) > 0 {
		// Log server SHOULD one instance.
		opt.ErrLog.Println("Log server is already running")
		return errGeneral
	}

	strg := storage.Storage{
		Root: storage.DirLayout{
			Root: opt.Conf.LogsDir(),
		},
	}
	if err := strg.Init(); err != nil {
		opt.ErrLog.Println("Failed to initialize the storage:", err)
		return errGeneral
	}

	if apiAddr == "" {
		apiAddr = config.DefaultApiServerAddr
	}
	if logAddr == "" {
		logAddr = config.DefaultLogServerAddr
	}

	simulatorStore := simulator.StateSimulatorStore{}

	// start API Server
	apiSrv := httpserver.NewHttpServer(apiAddr, restapi.NewRouter(restapi.RouterArgs{
		Config:         opt.Conf,
		Storage:        &strg,
		SimulatorStore: &simulatorStore,
	}))
	if err := apiSrv.Start(); err != nil {
		opt.ErrLog.Println("Failed to start the API server:", err)
		return errGeneral
	}
	defer func() {
		apiSrv.Stop()
		if err := apiSrv.Wait(); err != nil {
			opt.ErrLog.Println("Failed to stop the API server:", err)
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
		opt.ErrLog.Println("Failed to start the Log server:", err)
		return errGeneral
	}
	go logSrv.Serve()
	defer func() {
		defer logSrv.Wait()
		if err := logSrv.Close(); err != nil {
			opt.ErrLog.Println("Failed to stop the Log server:", err)
		}
	}()

	// add servers to config, and save
	opt.Conf.Servers.ApiServer[1] = &config.ApiServerConfig{
		ServerID: 1,
		Version:  1,
		Addr:     apiSrv.Url(),
	}
	opt.Conf.Servers.LogServer[1] = &config.LogServerConfig{
		ServerID: 1,
		Version:  1,
		Addr:     logSrv.ActualAddr(),
	}
	opt.Conf.WantSave()
	if err := opt.Conf.Save(); err != nil {
		opt.ErrLog.Println("Cannot write to the config file:", err)
		return errGeneral
	}

	// wait until a signal is received
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)
	<-sigCh

	// remove servers from config
	opt.Conf.Servers = *config.NewServers()
	opt.Conf.WantSave()
	if err := opt.Conf.Save(); err != nil {
		opt.ErrLog.Println("Cannot write to the config file:", err)
		return errGeneral
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

func getServerHandler(strg *storage.Storage, store *simulator.StateSimulatorStore) protocol.ServerHandler {
	m := &ServerHandlerMaker{
		Storage: strg,
		SSStore: store,
	}
	return m.ServerHandler()
}

type ServerHandlerMaker struct {
	Storage *storage.Storage
	SSStore *simulator.StateSimulatorStore

	initOnce sync.Once

	// chanの追加、削除、close()するときはLock()を、chanへの送受信はRLock()をかける。
	chMapLock sync.RWMutex

	// workerとの通信用。
	// close()されたら、workerは終了するべき。
	chMap map[protocol.ConnID]chan interface{}
}

func (m *ServerHandlerMaker) init() {
	m.initOnce.Do(func() {
		m.chMap = make(map[protocol.ConnID]chan interface{})
	})
}

func (m *ServerHandlerMaker) ServerHandler() protocol.ServerHandler {
	m.init()
	// TODO: コードを綺麗にする
	strg := m.Storage
	store := m.SSStore
	chMap := m.chMap
	chMapLock := &m.chMapLock

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

		// 最後にファイルと同期してから受信した RawFuncLog の個数。
		// flCount が flCountMax に達したら、ファイルに書き出す。
		var flCount int64
		const flCountMax = 1000000

		// 現在の状態をファイルに書き出す。
		// excludeRunning がtrueの場合、実行中のFuncLogは書き込まない。
		// writing フラグがtrueの場合は、何もしない。
		var writing int64
		writeCurrentState := func(excludeRunning bool) {
			if !atomic.CompareAndSwapInt64(&writing, 0, 1) {
				// 他のgoroutineが書き込んでいるため、今回は書き込みを行わない
				return
			}
			atomic.StoreInt64(&flCount, 0)
			logobj.FuncLog(func(store *storage.FuncLogStore) {
				for _, fl := range ss.FuncLogs(false) {
					if excludeRunning && !fl.IsEnded() {
						continue
					}
					err := store.SetNolock(fl)
					if err != nil {
						log.Panicln("ERROR: failed to append FuncLog during rotating:", err.Error())
					}
				}
			})
			logobj.Goroutine(func(store *storage.GoroutineStore) {
				for _, g := range ss.Goroutines() {
					err := store.SetNolock(g)
					if err != nil {
						log.Panicln("ERROR: failed to append Goroutine during rotating:", err.Error())
					}
				}
			})
			ss.Clear()
			if !atomic.CompareAndSwapInt64(&writing, 1, 0) {
				// writing フラグが立っていないのに書き込みを行ってしまった。
				panic("bug")
			}
		}
		// ログを閉じる前に、現在のStateSimulatorの状態を保存する。
		defer writeCurrentState(false)

		// StateSimulator の内容の書き出し要求を定期的に送信する。
		// chがcloseされたとき、タイミング次第でブロックされてしまう可能性がある。
		// そのため、このgoroutineの終了を待機しない。
		ssWriteReq := make(chan interface{})
		tick := time.NewTicker(1 * time.Second)
		defer tick.Stop()
		go func() {
			defer close(ssWriteReq)
			for range tick.C {
				ssWriteReq <- ss
			}
		}()

		for {
			var rawobj interface{}
			var ok bool
			select {
			case rawobj, ok = <-ch:
			case rawobj, ok = <-ssWriteReq:
			}
			if !ok {
				break
			}

			// lock獲得のオーバーヘッドを削減するため、チャンネルに次のitemがある場合は
			// lockを開放せずに連続して処理する。
			logobj.RawFuncLog(func(rawStore *storage.RawFuncLogStore) {
				defer func() {
					err = rawStore.FlushNolock()
					if err != nil {
						log.Panicln(err)
					}
				}()
				for {
					switch obj := rawobj.(type) {
					case *types.RawFuncLog:
						// NOTE: RawFuncLogが量がとても多いため、ストレージに書き込むと動作が遅くなってしまう。
						//       そのため、ファイルに書き出すのは止めた。
						//       コメントアウトすれば、デバッグするときに使えるかも?
						//if err := rawStore.SetNolock(obj); err != nil {
						//	log.Panicln("failed to append RawFuncLog:", err.Error())
						//}
						ss.Next(*obj)
						types.RawFuncLogPool.Put(obj)

						if atomic.AddInt64(&flCount, 1) >= flCountMax {
							// 多くの RawFuncLog をsimulatorに渡したため、大量のメモリを消費している。
							// ファイルに書き出してメモリを開放させる。
							writeCurrentState(false)
						}
					case *types.SymbolsData:
						if err := logobj.SetSymbolsData(obj); err != nil {
							log.Panicln("failed to append Symbols:", err.Error())
						}
					case *simulator.StateSimulator:
						writeCurrentState(false)
					default:
						log.Panicf("unsupported type: %+v", rawobj)
					}

					if len(ch) == 0 {
						break
					}
					rawobj = <-ch
				}
			})
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
		Symbols: func(id protocol.ConnID, s *types.SymbolsData) {
			chMapLock.RLock()
			chMap[id] <- s
			chMapLock.RUnlock()
		},
		RawFuncLog: func(id protocol.ConnID, f *types.RawFuncLog) {
			chMapLock.RLock()
			chMap[id] <- f
			chMapLock.RUnlock()
		},
	}
}
