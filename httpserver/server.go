package httpserver

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

type HttpServer struct {
	server *http.Server
	ctx    context.Context
	cancel context.CancelFunc
	sig    chan os.Signal
	errch  chan error
}

func NewHttpServer(addr string, args *ServerArgs) *HttpServer {
	router := getRouter(args)
	ctx, cancel := context.WithCancel(context.Background())

	return &HttpServer{
		server: &http.Server{
			Addr:    addr,
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
	if srv.server.Addr == "" {
		// find available port, and listen
		listener, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return err
		}
		srv.server.Addr = listener.Addr().String()
	} else {
		listener, err = net.Listen("tcp", srv.server.Addr)
		if err != nil {
			return err
		}
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
	srv.server.Shutdown(srv.ctx) // nolint: errcheck
	srv.cancel()                 // stop all handlers
}

func (srv *HttpServer) Addr() string {
	return srv.server.Addr
}
