package protocol

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/yuuki0xff/goapptrace/tracer/logutil"
	"github.com/yuuki0xff/xtcp"
)

const (
	DefaultTCPPort = 8600
	MaxListenTries = 100
)

// TCPコネクションを一意に識別するID
type ConnID int64

// サーバで発生したイベントのイベントハンドラ。
// 不要なフィールドはnilにすることが可能。
type ServerHandler struct {
	Connected    func(id ConnID)
	Disconnected func(id ConnID)

	Error func(id ConnID, err error)

	Symbols    func(id ConnID, diff *logutil.SymbolsDiff)
	RawFuncLog func(id ConnID, funclog *logutil.RawFuncLog)
}

// トレース対象との通信を行うサーバ。
// プロトコルの詳細は、README.mdに記載している。
//
// Usage:
//   srv.Listen()
//   go srv.Serve()
//
//   // wait for stop signal
//   time.Sleep(time.Second)
//
//   srv.Close()
//   srv.Wait()
type Server struct {
	// "unix:///path/to/socket/file" or "tcp://host:port"
	Addr    string
	Handler ServerHandler

	AppName         string
	Secret          string
	MaxBufferedMsgs int
	PingInterval    time.Duration

	listener net.Listener
	wg       sync.WaitGroup
	initOnce sync.Once
	stopOnce sync.Once

	connIDMap  map[*xtcp.Conn]ConnID
	nextConnID ConnID
	connIDLock sync.RWMutex

	connMap     map[ConnID]*ServerConn
	connMapLock sync.RWMutex

	opt     *xtcp.Options
	xtcpsrv *xtcp.Server
}

// Serverが管理しているコネクションの状態を管理と、イベントハンドラの呼び出しを行う。
type ServerConn struct {
	ID           ConnID
	Handler      *ServerHandler
	isNegotiated bool
}

func (s *Server) init() error {
	s.initOnce.Do(func() {
		if s.MaxBufferedMsgs <= 0 {
			s.MaxBufferedMsgs = DefaultMaxBufferedMsgs
		}
		if s.PingInterval == time.Duration(0) {
			s.PingInterval = DefaultPingInterval
		}
		s.connIDMap = map[*xtcp.Conn]ConnID{}
		s.connMap = map[ConnID]*ServerConn{}

		prt := &Proto{}
		s.opt = xtcp.NewOpts(s, prt)
		s.xtcpsrv = xtcp.NewServer(s.opt)
	})
	return nil
}

func (s *Server) Listen() error {
	if err := s.init(); err != nil {
		return err
	}

	var addr string
	var err error
	switch {
	case s.Addr == "":
		for i := 0; i < MaxListenTries; i++ {
			port := DefaultTCPPort + i
			addr = fmt.Sprintf("localhost:%d", port)
			s.Addr = "tcp://" + addr
			s.listener, err = net.Listen("tcp", addr)
			if err == nil {
				break
			}
		}
		if err != nil {
			return err
		}
	case strings.HasPrefix(s.Addr, "unix://"):
		// TODO: support unix domain socket
		return InvalidProtocolError
	case strings.HasPrefix(s.Addr, "tcp://"):
		addr = strings.TrimPrefix(s.Addr, "tcp://")
		s.listener, err = net.Listen("tcp", addr)
		if err != nil {
			return err
		}
	default:
		return InvalidProtocolError
	}
	return nil
}

func (s *Server) Serve() {
	s.wg.Add(1)
	defer s.wg.Done()

	s.xtcpsrv.Serve(s.listener)
}

func (s *Server) ActualAddr() string {
	addr := s.listener.Addr()
	return addr.Network() + "://" + addr.String()
}

func (s *Server) Close() error {
	s.stopOnce.Do(func() {
		// Stop method MUST NOT be called many times.
		s.xtcpsrv.Stop(xtcp.StopGracefullyAndWait)
	})
	return nil
}

func (s *Server) Wait() {
	s.wg.Wait()
}

func (s *Server) OnEvent(et xtcp.EventType, conn *xtcp.Conn, p xtcp.Packet) {
	connID := s.getConnID(conn)
	sc := s.getServerConn(connID)
	sc.OnEvent(et, conn, p)
}

// p will be nil when event is EventAccept/EventConnected/EventClosed
func (s *ServerConn) OnEvent(et xtcp.EventType, conn *xtcp.Conn, p xtcp.Packet) {
	switch et {
	case xtcp.EventAccept:
		// wait for client header packet to be received.
	case xtcp.EventRecv:
		if !s.isNegotiated {
			// check client header.
			pkt, ok := p.(*ClientHelloPacket)
			if !ok {
				conn.Stop(xtcp.StopImmediately)
				return
			}
			if !isCompatibleVersion(pkt.ProtocolVersion) {
				// 対応していないバージョンなら、切断する。
				conn.Stop(xtcp.StopImmediately)
				return
			}

			packet := &ServerHelloPacket{
				ProtocolVersion: ProtocolVersion,
			}
			if err := conn.Send(packet); err != nil {
				// TODO: try to reconnect
				panic(err)
			}
			s.isNegotiated = true

			if s.Handler.Connected != nil {
				s.Handler.Connected(s.ID)
			}
		} else {
			switch pkt := p.(type) {
			case *PingPacket:
				// do nothing
			case *ShutdownPacket:
				conn.Stop(xtcp.StopImmediately)
				return
			case *StartTraceCmdPacket:
				conn.Stop(xtcp.StopImmediately)
				return
			case *StopTraceCmdPacket:
				conn.Stop(xtcp.StopImmediately)
				return
			case *SymbolPacket:
				if s.Handler.Symbols != nil {
					s.Handler.Symbols(s.ID, &pkt.SymbolsDiff)
				}
			case *RawFuncLogPacket:
				if s.Handler.RawFuncLog != nil {
					s.Handler.RawFuncLog(s.ID, pkt.FuncLog)
				}
			default:
			}
		}
	case xtcp.EventSend:
	case xtcp.EventClosed:
		if s.Handler.Disconnected != nil {
			s.Handler.Disconnected(s.ID)
		}
	}
}

func (s *Server) getConnID(conn *xtcp.Conn) ConnID {
	s.connIDLock.Lock()
	defer s.connIDLock.Unlock()
	id, ok := s.connIDMap[conn]
	if ok {
		return id
	}

	id = s.nextConnID
	s.connIDMap[conn] = id
	s.nextConnID++
	return id
}

func (s *Server) getServerConn(id ConnID) *ServerConn {
	s.connMapLock.Lock()
	defer s.connMapLock.Unlock()
	srvConn, ok := s.connMap[id]
	if ok {
		return srvConn
	}

	srvConn = &ServerConn{
		ID:      id,
		Handler: &s.Handler,
	}
	s.connMap[id] = srvConn
	return srvConn
}
