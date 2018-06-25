package httpserver

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var (
	InvalidProtocolError = errors.New("invalid protocol")
)

// 任意のタイミングで終了可能なサーバ。
//
// Usage
//   srv := NewHttpServer("0.0.0.0:8080", ServerArgs{})
//   srv.Start()
//   go func() {
//   	time.Sleep(time.Second)
//   	srv.Stop()
//   }
//   err := srv.Wait()
type HttpServer struct {
	url    string
	server *http.Server
	ctx    context.Context
	cancel context.CancelFunc
	sig    chan os.Signal
	errch  chan error
}

func NewHttpServer(url string, router http.Handler) *HttpServer {
	ctx, cancel := context.WithCancel(context.Background())

	return &HttpServer{
		url: url,
		server: &http.Server{
			Handler: router,
		},
		ctx:    ctx,
		cancel: cancel,
		sig:    make(chan os.Signal),
		errch:  make(chan error),
	}
}

// HTTPサーバといくつかのハンドラを起動する。
// この関数の実行終了後、Addr()から実際にlistenされたアドレスとポート番号を取得出来る。
func (srv *HttpServer) Start() error {
	// start a signal handler
	signal.Notify(srv.sig, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	go func() {
		select {
		case <-srv.sig:
			break
		case <-srv.ctx.Done():
			break
		}
		signal.Stop(srv.sig)
		srv.Stop()
	}()

	var listener net.Listener
	var err error
	switch {
	case strings.HasPrefix(srv.url, "http://"):
		srv.server.Addr = strings.TrimPrefix(srv.url, "http://")
		listener, err = net.Listen("tcp", srv.server.Addr)
		if err != nil {
			return err
		}
	default:
		return InvalidProtocolError
	}

	go func() {
		defer srv.Stop()       // stop all handlers
		defer listener.Close() // nolint
		srv.errch <- srv.server.Serve(listener)
	}()
	return nil
}

// HTTPサーバが終了するまで待機する。
func (srv *HttpServer) Wait() error {
	err := <-srv.errch
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// HTTPサーバと全てのハンドラを停止する。
func (srv *HttpServer) Stop() {
	srv.server.Shutdown(srv.ctx) // nolint: errcheck, gas
	srv.cancel()                 // stop all handlers
}

// Addr returns binded address of this server.
func (srv *HttpServer) Addr() string {
	return srv.server.Addr
}

// Url returns complete URL with prefix "http://".
func (srv *HttpServer) Url() string {
	return "http://" + srv.Addr()
}
