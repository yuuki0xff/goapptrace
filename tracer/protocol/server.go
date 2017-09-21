package protocol

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"net"
	"strings"

	"time"

	"reflect"

	"sync"

	"github.com/yuuki0xff/goapptrace/tracer/log"
)

type ServerHandler struct {
	Connected    func()
	Disconnected func()

	Error func(error)

	Symbols    func(*log.Symbols)
	RawFuncLog func(*log.RawFuncLogNew)
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
	workerWg  sync.WaitGroup

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

	s.workerWg.Add(1)
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
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil

		close(s.writeChan)

		s.workerWg.Wait()
		err := s.listener.Close()
		s.listener = nil
		s.Handler.Disconnected()
		return err
	}
	return nil
}

func (s *Server) Wait() {
	s.workerWg.Wait()
}

func (s *Server) worker() {
	defer s.workerWg.Done()
	errCh := make(chan error)
	shouldStop := false

	isErrorNoStop := func(err error) bool {
		if err != nil {
			if isEOF(err) || isBrokenPipe(err) {
				// ignore errors
				errCh <- nil
				return true
			}
			errCh <- err
			return true
		}
		return false
	}
	isError := func(err error) bool {
		if shouldStop {
			return true
		}
		return isErrorNoStop(err)
	}

	handleConn := func(conn net.Conn) {
		defer s.workerWg.Done()
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

		println("server: read client header")
		setReadDeadline()
		clientHeader := ClientHeader{}
		if isErrorNoStop(dec.Decode(&clientHeader)) {
			return
		}
		println("server: read client header done")
		// TODO: check response

		println("server: send server header")
		setWriteDeadline()
		if isErrorNoStop(enc.Encode(&ServerHeader{
			ServerVersion: "", //TODO
		})) {
			return
		}
		println("server: send server header done")

		// initialize process is done
		// start read/write workers
		println("server: initialize done")

		// start read worker
		s.workerWg.Add(1)
		go func() {
			defer s.workerWg.Done()
			for !shouldStop {
				setReadDeadline()
				msgHeader := &MessageHeader{}
				if isError(dec.Decode(msgHeader)) {
					return
				}

				var data interface{}
				switch msgHeader.MessageType {
				case PingMsg:
					data = &PingMsgData{}
				case ShutdownMsg:
					data = &ShutdownMsgData{}
				case SymbolsMsg:
					data = &log.Symbols{}
				case RawFuncLogMsg:
					data = &log.RawFuncLogNew{}
				default:
					errCh <- errors.New(fmt.Sprintf("Invalid MessageType: %d", msgHeader.MessageType))
					return
				}

				setReadDeadline()
				if isErrorNoStop(dec.Decode(data)) {
					return
				}
				fmt.Printf("server data: %s : %+v\n", reflect.TypeOf(data).String(), data)

				switch msgHeader.MessageType {
				case PingMsg:
					// do nothing
				case ShutdownMsg:
					// do nothing
				case SymbolsMsg:
					s.Handler.Symbols(data.(*log.Symbols))
				case RawFuncLogMsg:
					s.Handler.RawFuncLog(data.(*log.RawFuncLogNew))
				default:
					panic("bug")
				}
			}
		}()

		// start ping worker
		s.workerWg.Add(1)
		go func() {
			defer s.workerWg.Done()
			for !shouldStop {
				s.Send(PingCmd, &PingCmdArgs{})
				time.Sleep(s.PingInterval)
			}
		}()

		// start write worker
		s.workerWg.Add(1)
		go func() {
			defer s.workerWg.Done()
			// will be closing c.writeChan by c.Close() when occurred shutdown request.
			// so, this worker should not check 'shouldStop' variable.
			for data := range s.writeChan {
				setWriteDeadline()
				if isError(enc.Encode(data)) {
					return
				}
			}
		}()
	}

	s.workerWg.Add(1)
	go func() {
		defer s.workerWg.Done()
		for !shouldStop {
			conn, err := s.listener.Accept()
			if isError(err) {
				return
			}
			s.workerWg.Add(1)
			go handleConn(conn)
		}
	}()

	select {
	case <-s.workerCtx.Done():
		// do nothing
	case err := <-errCh:
		if err != nil {
			s.Handler.Error(err)
		}
	}
	shouldStop = true
}
