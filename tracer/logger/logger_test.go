package logger

import (
	"os"
	"strings"
	"testing"

	"reflect"

	"path/filepath"

	"github.com/yuuki0xff/goapptrace/info"
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
	checkLogServerSender(t)

	if !connected {
		t.Fatal("connected should true, but false")
	}

	srv.Close()
	if !disconnected {
		t.Fatal("disconnected should true, but false")
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
	if !(strings.HasPrefix(fpath, prefix) && strings.HasSuffix(fpath, ".log")) {
		t.Fatalf("invalid output file fpath: %s", fpath)
	}

	// check close
	Close()
	if sender != nil {
		t.Fatalf("sender should nil, but %+v", sender)
	}
}

func checkLogServerSender(t *testing.T) {
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

	// check close
	Close()
	if sender != nil {
		t.Fatalf("sender should nil, but %+v", sender)
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
