package logger

import (
	"fmt"
	"log"
	"os"

	"github.com/yuuki0xff/goapptrace/info"
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
	"github.com/yuuki0xff/goapptrace/tracer/protocol"
)

// LogServerSender sends Symbols and FuncLog to the log server
type LogServerSender struct {
	client *protocol.Client
}

// LogServerSenderが使用できる場合はtrueを返す。
func CanUseLogServerSender() bool {
	_, ok := os.LookupEnv(info.DEFAULT_LOGSRV_ENV)
	return ok
}

// サーバとのセッションを確立する。
// セッションが確立するまで処理をブロックする。
func (s *LogServerSender) Open() error {
	url, ok := os.LookupEnv(info.DEFAULT_LOGSRV_ENV)
	if !ok {
		return fmt.Errorf("not found %s environment value", info.DEFAULT_LOGSRV_ENV)
	}

	s.client = &protocol.Client{
		Addr: url,
		Handler: protocol.ClientHandler{
			Connected:    func() {},
			Disconnected: func() {},
			Error: func(err error) {
				fmt.Println("s.client ERROR:", err.Error())
			},
			StartTrace: func(args *protocol.StartTraceCmdPacket) {},
			StopTrace:  func(args *protocol.StopTraceCmdPacket) {},
		},
		AppName: "TODO", // TODO
		Secret:  "secret",
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

// send Symbols and RawFuncLog to the log server.
func (s *LogServerSender) Send(diff *logutil.SymbolsData, funclog *logutil.RawFuncLog) error {
	if s.client == nil {
		return ClosedError
	}

	if diff != nil {
		if err := s.client.Send(&protocol.SymbolPacket{
			SymbolsData: *diff,
		}); err != nil {
			return err
		}
	}
	if funclog != nil {
		if err := s.client.Send(&protocol.RawFuncLogPacket{
			FuncLog: funclog,
		}); err != nil {
			return err
		}
	}
	return nil
}
