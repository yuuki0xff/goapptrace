package logger

import (
	"os"
	"strings"
	"testing"

	"reflect"

	"path/filepath"

	"time"

	"github.com/yuuki0xff/goapptrace/info"
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
	"github.com/yuuki0xff/goapptrace/tracer/protocol"
)

func TestSetOutput_writeToFile_useDefaultPrefix(t *testing.T) {
	os.Unsetenv(info.DEFAULT_LOGSRV_ENV)
	os.Unsetenv(info.DEFAULT_LOGFILE_ENV)

	abspath, err := filepath.Abs(info.DEFAULT_LOGFILE_PREFIX)
	if err != nil {
		t.Fatal(err)
	}
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
	os.Setenv(info.DEFAULT_LOGFILE_ENV, "/tmp/.goapptrace-logger-test")
	sender := &RetrySender{
		Sender:        &FileSender{},
		MaxRetry:      defaultMaxRetry,
		RetryInterval: defaultRetryInterval,
	}

	if err := sender.Open(); err != nil {
		t.Fatalf("failed to sender.Open(): %s", err)
	}

	// send a log.
	if err := sender.Send(
		&logutil.Symbols{
			Funcs: []*logutil.FuncSymbol{
				{logutil.FuncID(0), "module.f1", "/go/src/module/src.go", 1},
				{logutil.FuncID(1), "module.f2", "/go/src/module/src.go", 2},
			},
			FuncStatus: []*logutil.FuncStatus{
				{logutil.FuncStatusID(0), logutil.FuncID(0), 10, 100},
				{logutil.FuncStatusID(1), logutil.FuncID(1), 20, 200},
			},
		},
		&logutil.RawFuncLog{
			ID:        logutil.RawFuncLogID(0),
			Tag:       "funcStart",
			Timestamp: logutil.NewTime(time.Now()),
			Frames:    []logutil.FuncStatusID{0, 1},
		},
	); err != nil {
		t.Fatalf("fialed to sender.Send(): %s", err)
	}

	// will be occur the send error. but RetrySender will handle error, and try to recovery.
	// so sender.Send() will return the nil.
	if err := sender.Sender.Close(); err != nil {
		t.Fatalf("failed to FileSender.Close(): %s", err)
	}
	if err := sender.Send(
		&logutil.Symbols{
			Funcs: []*logutil.FuncSymbol{},
			FuncStatus: []*logutil.FuncStatus{
				{logutil.FuncStatusID(2), logutil.FuncID(1), 21, 210},
			},
		},
		&logutil.RawFuncLog{
			ID:        logutil.RawFuncLogID(1),
			Tag:       "funcEnd",
			Timestamp: logutil.NewTime(time.Now()),
			Frames:    []logutil.FuncStatusID{0, 2},
		},
	); err != nil {
		t.Fatalf("failed to error recovery on sender.Send(): %s", err)
	}
	if err := sender.Close(); err != nil {
		t.Fatalf("failed to sender.Close(): %s", err)
	}
}

func checkFileSender(t *testing.T, prefix string) {
	setOutput()

	// check sender type
	retrySender, ok := sender.(*RetrySender)
	if !ok {
		t.Fatalf("mismatch type: expect=*RetrySender actual=%s", reflect.TypeOf(sender))
	}
	fileSender, ok := retrySender.Sender.(*FileSender)
	if !ok {
		t.Fatalf("mismatch type: expect=*FileSender actual=%s", reflect.TypeOf(sender))
	}

	// check file path
	fpath := fileSender.logFilePath()
	os.Remove(fpath)
	if !(strings.HasPrefix(fpath, prefix) && strings.HasSuffix(fpath, ".log.gz")) {
		t.Fatalf("invalid output file fpath: %s", fpath)
	}

	// check sendLog()
	sendLog(logutil.FuncStart, logutil.TxID(0))
	sendLog(logutil.FuncStart, logutil.TxID(1))
	sendLog(logutil.FuncEnd, logutil.TxID(2))
	sendLog(logutil.FuncEnd, logutil.TxID(3))

	// check close
	Close()
	if sender != nil {
		t.Fatalf("sender should nil, but %+v", sender)
	}
}

func checkLogServerSender(t *testing.T, connected, disconnected *bool) {
	setOutput()

	// check sender type
	retrySender, ok := sender.(*RetrySender)
	if !ok {
		t.Fatalf("mismatch type: expect=*RetrySender actual=%s", reflect.TypeOf(sender))
	}
	_, ok = retrySender.Sender.(*LogServerSender)
	if !ok {
		t.Fatalf("mismatch type: expect=*LogServerSender actual=%s", reflect.TypeOf(sender))
	}

	// check sendLog()
	sendLog(logutil.FuncStart, logutil.TxID(0))
	sendLog(logutil.FuncStart, logutil.TxID(1))
	sendLog(logutil.FuncEnd, logutil.TxID(2))
	sendLog(logutil.FuncEnd, logutil.TxID(3))

	// is handled Connected event?
	if !*connected {
		t.Fatal("connected should true, but false")
	}

	// check close
	Close()
	if sender != nil {
		t.Fatalf("sender should nil, but %+v", sender)
	}

	// is handled Disconnected event until 1000 milliseconds?
	for i := 0; i < 100; i++ {
		if *disconnected {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !*disconnected {
		t.Fatal("disconnected should true, but false")
	}
}

func startLogServer(t *testing.T, connected, disconnected *bool) *protocol.Server {
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
	if err := srv.Listen(); err != nil {
		t.Fatalf("LogServer can not listen: %s", err)
	}
	go srv.Serve()
	return srv
}
