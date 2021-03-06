package logger

import (
	"fmt"
	"log"
	"os"

	"github.com/yuuki0xff/goapptrace/info"
	"github.com/yuuki0xff/goapptrace/tracer/protocol"
	"github.com/yuuki0xff/goapptrace/tracer/types"
)

// LogServerSender sends Symbols and FuncLog to the log server
type LogServerSender struct {
	client *protocol.Client
}

// LogServerSenderが使用できる場合はtrueを返す。
func CanUseLogServerSender() bool {
	_, ok := os.LookupEnv(info.DefaultLogsrvEnv)
	return ok
}

// サーバとのセッションを確立する。
// セッションが確立するまで処理をブロックする。
func (s *LogServerSender) Open() error {
	url, ok := os.LookupEnv(info.DefaultLogsrvEnv)
	if !ok {
		return fmt.Errorf("not found %s environment value", info.DefaultLogsrvEnv)
	}

	hostname, _ := os.Hostname()
	appname := os.Getenv(info.DefaultAppNameEnv)
	if appname == "" {
		appname = os.Args[0]
	}
	s.client = &protocol.Client{
		Addr: url,
		Handler: protocol.ClientHandler{
			Connected:    func() {},
			Disconnected: func() {},
			Error: func(err error) {
				fmt.Println("s.client ERROR:", err.Error())
			},
			StartTrace: func(pkt *protocol.StartTraceCmdPacket) {
				if pkt.FuncName != "" {
					EnableTrace(pkt.FuncName)
				} else {
					panic("FuncName MUST NOT empty")
				}
			},
			StopTrace: func(pkt *protocol.StopTraceCmdPacket) {
				if pkt.FuncName != "" {
					DisableTrace(pkt.FuncName)
				} else {
					panic("FuncName MUST NOT empty")
				}
			},
		},
		PID:     uint64(os.Getpid()),
		AppName: appname,
		Host:    hostname,
		Secret:  "secret", // TODO
	}
	s.client.Init()
	go func() {
		if err := s.client.Serve(); err != nil {
			log.Panic(err)
		}
	}()
	s.client.WaitNegotiation()
	return nil
}

// サーバとのセッションを切る。
// 正常終了するまで処理をブロックする。
func (s *LogServerSender) Close() error {
	if s.client == nil {
		return ClosedError
	}

	if err := s.client.Close(); err != nil {
		return err
	}
	s.client = nil
	return nil
}

// send Symbols to the log server.
func (s *LogServerSender) SendSymbols(data *types.SymbolsData) error {
	// SymbolPacketは非常に大きくなる可能性が高いため、SendLargeを使って送信する。
	return s.client.SendLarge(&protocol.SymbolPacket{
		SymbolsData: *data,
	})
}

// send RawFuncLog to the log server.
func (s *LogServerSender) SendLog(raw *types.RawFuncLog) error {
	if s.client == nil {
		return ClosedError
	}

	if err := s.client.Send(&protocol.RawFuncLogPacket{
		FuncLog: raw,
	}); err != nil {
		return err
	}
	return nil
}
