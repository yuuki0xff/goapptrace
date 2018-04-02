package protocol

import (
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/yuuki0xff/goapptrace/tracer/types"
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
	Connected    func()
	Disconnected func()

	Error func(err error)

	Symbols    func(diff *types.SymbolsData)
	RawFuncLog func(funclog *types.RawFuncLog)
}

// SetDefault sets "fn" to all nil fields.
func (sh ServerHandler) SetDefault(fn func(field string)) ServerHandler {
	if sh.Connected == nil {
		sh.Connected = func() {
			fn("Connected")
		}
	}
	if sh.Disconnected == nil {
		sh.Disconnected = func() {
			fn("Disconnected")
		}
	}
	if sh.Error == nil {
		sh.Error = func(err error) {
			fn("Error")
		}
	}
	if sh.Symbols == nil {
		sh.Symbols = func(diff *types.SymbolsData) {
			fn("Symbols")
		}
	}
	if sh.RawFuncLog == nil {
		sh.RawFuncLog = func(funclog *types.RawFuncLog) {
			fn("RawFuncLog")
		}
	}
	return sh
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
	Addr       string
	NewHandler func(id ConnID) *ServerHandler

	AppName      string
	Secret       string
	PingInterval time.Duration
	BufferOpt    BufferOption

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
	sendHandler  func(conn *xtcp.Conn, packet xtcp.Packet)
	stopHandler  func(conn *xtcp.Conn, mode xtcp.StopMode)
}

func (s *Server) init() error {
	s.initOnce.Do(func() {
		if s.PingInterval == time.Duration(0) {
			s.PingInterval = DefaultPingInterval
		}
		s.BufferOpt.SetDefault()
		s.connIDMap = map[*xtcp.Conn]ConnID{}
		s.connMap = map[ConnID]*ServerConn{}

		prt := &Proto{}
		s.opt = xtcp.NewOpts(s, prt)
		s.BufferOpt.Xtcp.Set(s.opt)
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
				s.error(fmt.Errorf("negotiation failed: client sends an unexpected packet: %#v", p))
				s.stop(conn, xtcp.StopImmediately)
				return
			}
			if !isCompatibleVersion(pkt.ProtocolVersion) {
				// 対応していないバージョンなら、切断する。
				s.error(fmt.Errorf("negotiation failed: client version is not compatible"))
				s.stop(conn, xtcp.StopImmediately)
				return
			}

			packet := &ServerHelloPacket{
				ProtocolVersion: ProtocolVersion,
			}
			err := s.send(conn, packet)
			if err != nil {
				s.error(err)
				s.stop(conn, xtcp.StopImmediately)
				return
			}

			s.isNegotiated = true
			if s.Handler.Connected != nil {
				s.Handler.Connected()
			}
		} else {
			switch pkt := p.(type) {
			case *PingPacket:
				// do nothing
			case *SymbolPacket:
				if s.Handler.Symbols != nil {
					s.Handler.Symbols(&pkt.SymbolsData)
				}
			case *RawFuncLogPacket:
				if s.Handler.RawFuncLog != nil {
					s.Handler.RawFuncLog(pkt.FuncLog)
				}
			default:
				s.error(fmt.Errorf("server receives an unexpected packet: %#v", pkt))
				s.stop(conn, xtcp.StopImmediately)
				return
			}
		}
	case xtcp.EventSend:
	case xtcp.EventClosed:
		if s.Handler.Disconnected != nil {
			s.Handler.Disconnected()
		}
	}
}
func (s *ServerConn) send(conn *xtcp.Conn, p xtcp.Packet) error {
	if s.sendHandler == nil {
		return conn.Send(p)
	} else {
		// call the mock method.
		s.sendHandler(conn, p)
		return nil
	}
}

func (s *ServerConn) stop(conn *xtcp.Conn, mode xtcp.StopMode) {
	if s.stopHandler == nil {
		conn.Stop(mode)
	} else {
		// call the mock method.
		s.stopHandler(conn, mode)
		// "xtcp.EventClosed" event occurs after calling the conn.Stop().
		// But the event is not occur when calling s.stopHandler().
		// So we occurs the event here.
		s.OnEvent(xtcp.EventClosed, conn, nil)
	}
}

func (s *ServerConn) error(err error) {
	if s.Handler.Error != nil {
		s.Handler.Error(err)
	} else {
		log.Println("ERROR: ", err)
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
		Handler: s.NewHandler(id),
	}
	s.connMap[id] = srvConn
	return srvConn
}
