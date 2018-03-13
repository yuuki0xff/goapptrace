package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yuuki0xff/goapptrace/info"
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
	"github.com/yuuki0xff/goapptrace/tracer/protocol"
	"github.com/yuuki0xff/goapptrace/tracer/types"
)

func TestSetOutput_writeToFile_useDefaultPrefix(t *testing.T) {
	a := assert.New(t)
	os.Unsetenv(info.DEFAULT_LOGSRV_ENV)
	os.Unsetenv(info.DEFAULT_LOGFILE_ENV)

	abspath, err := filepath.Abs(info.DEFAULT_LOGFILE_PREFIX)
	a.NoError(err)
	checkFileSender(t, abspath)
}

func TestSetOutput_writeToFile_usePrefix(t *testing.T) {
	os.Unsetenv(info.DEFAULT_LOGSRV_ENV)
	os.Setenv(info.DEFAULT_LOGFILE_ENV, "/tmp/.goapptrace-logger-test")

	checkFileSender(t, "/tmp/.goapptrace-logger-test.")
}

func TestSetOutput_connectToLogServer(t *testing.T) {
	var connected bool
	var disconnected bool

	srv := startLogServer(t, &connected, &disconnected)
	os.Setenv(info.DEFAULT_LOGSRV_ENV, srv.ActualAddr())
	checkLogServerSender(t, &connected, &disconnected)
}

func TestRetrySender(t *testing.T) {
	a := assert.New(t)
	os.Setenv(info.DEFAULT_LOGFILE_ENV, "/tmp/.goapptrace-logger-test")
	sender := &RetrySender{
		Sender:        &FileSender{},
		MaxRetry:      defaultMaxRetry,
		RetryInterval: defaultRetryInterval,
	}

	a.NoError(sender.Open())

	// send a log.
	a.NoError(sender.Send(
		&types.SymbolsData{
			Funcs: []*types.GoFunc{
				{types.FuncID(0), "module.f1", "/go/src/module/src.go", 1},
				{types.FuncID(1), "module.f2", "/go/src/module/src.go", 2},
			},
			Lines: []*types.GoLine{
				{types.GoLineID(0), logutil.FuncID(0), 10, 100},
				{types.GoLineID(1), logutil.FuncID(1), 20, 200},
			},
		},
		&types.RawFuncLog{
			ID:        types.RawFuncLogID(0),
			Tag:       "funcStart",
			Timestamp: types.NewTime(time.Now()),
			Frames:    []types.GoLineID{0, 1},
		},
	))

	// will be occur the send error. but RetrySender will handle error, and try to recovery.
	// so sender.Send() will return the nil.
	a.NoError(sender.Sender.Close())
	a.NoError(sender.Send(
		&types.SymbolsData{
			Funcs: []*types.GoFunc{},
			Lines: []*types.GoLine{
				{types.GoLineID(2), logutil.FuncID(1), 21, 210},
			},
		},
		&types.RawFuncLog{
			ID:        types.RawFuncLogID(1),
			Tag:       "funcEnd",
			Timestamp: types.NewTime(time.Now()),
			Frames:    []types.GoLineID{0, 2},
		},
	))

	a.NoError(sender.Close())
}

func checkFileSender(t *testing.T, prefix string) {
	a := assert.New(t)
	setOutput()

	// check sender type
	a.IsType(&RetrySender{}, sender)
	retrySender := sender.(*RetrySender)
	a.IsType(&FileSender{}, retrySender.Sender)
	fileSender := retrySender.Sender.(*FileSender)

	// check file path
	fpath := fileSender.logFilePath()
	os.Remove(fpath)
	a.Truef(strings.HasPrefix(fpath, prefix), "invalid output file fpath: %s", fpath)
	a.Truef(strings.HasSuffix(fpath, ".log.gz"), "invalid output file fpath: %s", fpath)

	// check sendLog()
	sendLog(types.FuncStart, logutil.TxID(0))
	sendLog(types.FuncStart, logutil.TxID(1))
	sendLog(types.FuncEnd, logutil.TxID(2))
	sendLog(types.FuncEnd, logutil.TxID(3))

	// check close
	Close()
	a.Nil(sender)
}

func checkLogServerSender(t *testing.T, connected, disconnected *bool) {
	a := assert.New(t)
	setOutput()

	// check sender type
	a.IsType(&RetrySender{}, sender)
	retrySender := sender.(*RetrySender)
	a.IsType(&LogServerSender{}, retrySender.Sender)
	_ = retrySender.Sender.(*LogServerSender)

	// check sendLog()
	sendLog(types.FuncStart, logutil.TxID(0))
	sendLog(types.FuncStart, logutil.TxID(1))
	sendLog(types.FuncEnd, logutil.TxID(2))
	sendLog(types.FuncEnd, logutil.TxID(3))

	// is handled Connected event?
	a.True(*connected)

	// check close
	Close()
	a.Nil(sender)

	// is handled Disconnected event until 1000 milliseconds?
	for i := 0; i < 100; i++ {
		if *disconnected {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	a.True(*disconnected)
}

func startLogServer(t *testing.T, connected, disconnected *bool) *protocol.Server {
	a := assert.New(t)
	srv := &protocol.Server{
		Addr: "",
		Handler: protocol.ServerHandler{
			Connected: func(id protocol.ConnID) {
				*connected = true
			},
			Disconnected: func(id protocol.ConnID) {
				*disconnected = true
			},
			Error: func(id protocol.ConnID, err error) {
				t.Fatalf("An error occurred in LogServer: %s", err)
			},
		},
		AppName: "goapptrace-logger-test",
		Secret:  "secret",
	}
	a.NoError(srv.Listen())
	go srv.Serve()
	return srv
}
