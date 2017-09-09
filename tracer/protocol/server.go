package protocol

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"net"
	"strings"

	"time"

	"github.com/yuuki0xff/goapptrace/tracer/log"
)

type ServerHandler struct {
	Connected    func()
	Disconnected func()

	Error func(error)

	Symbols func(*log.Symbols)
	FuncLog func(*log.FuncLog)
}

type Server struct {
	// "unix:///path/to/socket/file" or "tcp://host:port"
	Addr    string
	Handler ServerHandler

	AppName         string
	Version         string
	Secret          string
	MaxBufferedMsgs int
	Timeout         time.Duration
	PingInterval    time.Duration

	listener  net.Listener
	cancel    context.CancelFunc
	workerCtx context.Context

	writeChan chan interface{}
}

func (s *Server) Listen() error {
	var proto string
	var url string
	var err error

	switch {
	case strings.HasPrefix(s.Addr, "unix://"):
		url = strings.TrimPrefix(s.Addr, "unix://")
		proto = "unix"
	case strings.HasPrefix(s.Addr, "tcp://"):
		url = strings.TrimPrefix(s.Addr, "tcp://")
		proto = "tcp"
	default:
		return errors.New("Invalid protocol")
	}

	s.listener, err = net.Listen(proto, url)
	if err != nil {
		return err
	}

	s.workerCtx, s.cancel = context.WithCancel(context.Background())
	if s.MaxBufferedMsgs <= 0 {
		s.MaxBufferedMsgs = DefaultMaxBufferedMsgs
	}
	s.writeChan = make(chan interface{}, s.MaxBufferedMsgs)
	if s.Timeout == time.Duration(0) {
		s.Timeout = DefaultTimeout
	}
	if s.PingInterval == time.Duration(0) {
		s.PingInterval = DefaultPingInterval
	}

	go s.worker()
	s.Handler.Connected()
	return nil
}

func (s *Server) ActualAddr() string {
	addr := s.listener.Addr()
	return addr.Network() + "://" + addr.String()
}

func (s *Server) Send(cmdType CommandType, args interface{}) {
	s.writeChan <- &CommandHeader{
		CommandType: cmdType,
	}
	s.writeChan <- args
}

func (s *Server) Close() error {
	if s.workerCtx != nil {
		s.workerCtx = nil
		s.cancel()
		s.cancel = nil

		close(s.writeChan)

		err := s.listener.Close()
		s.listener = nil
		s.Handler.Disconnected()
		return err
	}
	return nil
}

func (s *Server) Wait() {
	if s.workerCtx != nil {
		<-s.workerCtx.Done()
	}
}

func (s *Server) worker() {
	errCh := make(chan error)
	shouldStop := false

	isError := func(err error) bool {
		if shouldStop {
			return true
		}
		if err != nil {
			errCh <- err
			return true
		}
		return false
	}

	handleConn := func(conn net.Conn) {
		setReadDeadline := func() {
			if err := conn.SetReadDeadline(time.Now().Add(s.Timeout)); err != nil {
				panic(err)
			}
		}
		setWriteDeadline := func() {
			if err := conn.SetWriteDeadline(time.Now().Add(s.Timeout)); err != nil {
				panic(err)
			}
		}

		// initialize
		enc := gob.NewEncoder(conn)
		dec := gob.NewDecoder(conn)

		setReadDeadline()
		clientHeader := ClientHeader{}
		if isError(dec.Decode(&clientHeader)) {
			return
		}
		// TODO: check response

		setWriteDeadline()
		if isError(enc.Encode(&ServerHeader{
			ServerVersion: "", //TODO
		})) {
			return
		}

		// initialize process is done
		// start read/write workers

		// start read worker
		go func() {
			for !shouldStop {
				setReadDeadline()
				msgHeader := &MessageHeader{}
				if isError(dec.Decode(msgHeader)) {
					return
				}
				if shouldStop {
					return
				}

				var data interface{}
				switch msgHeader.MessageType {
				case PingMsg:
					data = &PingMsgData{}
				case SymbolsMsg:
					data = &log.Symbols{}
				case FuncLogMsg:
					data = &log.FuncLog{}
				default:
					errCh <- errors.New(fmt.Sprintf("Invalid MessageType: %d", msgHeader.MessageType))
					return
				}

				setReadDeadline()
				if isError(dec.Decode(data)) {
					return
				}
				if shouldStop {
					return
				}

				switch msgHeader.MessageType {
				case PingMsg:
					// do nothing
				case SymbolsMsg:
					s.Handler.Symbols(data.(*log.Symbols))
				case FuncLogMsg:
					s.Handler.FuncLog(data.(*log.FuncLog))
				default:
					panic("bug")
				}
			}
		}()

		// start ping worker
		go func() {
			for !shouldStop {
				s.Send(PingCmd, &PingCmdArgs{})
				time.Sleep(s.PingInterval)
			}
		}()

		// start write worker
		go func() {
			for data := range s.writeChan {
				if shouldStop {
					return
				}

				setWriteDeadline()
				if isError(enc.Encode(data)) {
					return
				}
			}
		}()
	}

	go func() {
		for !shouldStop {
			conn, err := s.listener.Accept()
			if isError(err) {
				return
			}
			go handleConn(conn)
		}
	}()

	select {
	case <-s.workerCtx.Done():
		shouldStop = true
		return
	case err := <-errCh:
		s.Handler.Error(err)
		shouldStop = true
		if err := s.Close(); err != nil {
			s.Handler.Error(err)
		}
		return
	}
}
