package logger

import (
	"os"
	"strings"
	"testing"

	"github.com/yuuki0xff/goapptrace/info"
	"github.com/yuuki0xff/goapptrace/tracer/protocol"
)

func TestSetOutput_writeToFile_useDefaultPrefix(t *testing.T) {
	os.Unsetenv(info.DEFAULT_LOGSRV_ENV)
	os.Unsetenv(info.DEFAULT_LOGFILE_ENV)
	setOutput()
	if OutputFile == nil {
		t.Fatal("OutputFile should not nil, but nil")
	}
	if Client != nil {
		t.Fatalf("Client should nil, but %+v", Client)
	}
	name := OutputFile.Name()
	os.Remove(name)
	if !(strings.HasPrefix(name, info.DEFAULT_LOGFILE_PREFIX) && strings.HasSuffix(name, ".log")) {
		t.Fatalf("Invalid output file name: %s", OutputFile.Name())
	}
	Close()
}

func TestSetOutput_writeToFile_usePrefix(t *testing.T) {
	os.Unsetenv(info.DEFAULT_LOGSRV_ENV)
	os.Setenv(info.DEFAULT_LOGFILE_ENV, "/tmp/.goapptrace-logger-test")

	setOutput()
	if OutputFile == nil {
		t.Fatal("OutputFile should not nil, but nil")
	}
	if Client != nil {
		t.Fatalf("Client should nil, but %+v", Client)
	}
	name := OutputFile.Name()
	os.Remove(name)
	if !(strings.HasPrefix(name, "/tmp/.goapptrace-logger-test.") && strings.HasSuffix(name, ".log")) {
		t.Fatalf("Invalid output file name: %s", OutputFile.Name())
	}
	Close()
}

func TestSetOutput_connectToLogServer(t *testing.T) {
	var connected bool
	var disconnected bool

	// start log server
	srv := protocol.Server{
		Addr: "",
		Handler: protocol.ServerHandler{
			Connected: func(id protocol.ConnID) {
				connected = true
			},
			Disconnected: func(id protocol.ConnID) {
				disconnected = true
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
	defer srv.Close()
	go srv.Serve()

	os.Setenv(info.DEFAULT_LOGSRV_ENV, srv.ActualAddr())
	setOutput()
	if OutputFile != nil {
		t.Fatalf("OutputFile should nil, but %+v", OutputFile)
	}
	if Client == nil {
		t.Fatal("Client should not nil, but nil")
	}
	if Client.Addr != srv.ActualAddr() {
		t.Fatalf("Mismatch address: client.Addr=%s server.ActualAddr=%s", Client.Addr, srv.ActualAddr())
	}
	Close()
}
