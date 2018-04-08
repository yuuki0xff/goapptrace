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

type PacketSender interface {
	Send(p xtcp.Packet) error
	Stop(mode xtcp.StopMode)
}

// TCPコネクションで発生したイベントのハンドラ。
// 1つのコネクションごとにハンドラが作成されるため、handlerの中でconnection localな変数を保持しても構わない。
// 不要なフィールドはnilにすることが可能。
type ConnHandler struct {
	Connected    func(pkt *ClientHelloPacket)
	Disconnected func()

	Error func(err error)

	Symbols    func(diff *types.SymbolsData)
	RawFuncLog func(funclog *types.RawFuncLog)
}

// SetDefault sets "fn" to all nil fields.
func (sh ConnHandler) SetDefault(fn func(field string)) ConnHandler {
	if sh.Connected == nil {
		sh.Connected = func(pkt *ClientHelloPacket) {
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
	NewHandler func(id ConnID, conn PacketSender) *ConnHandler

	AppName      string
	Secret       string
	PingInterval time.Duration
	BufferOpt    BufferOption

	listener net.Listener
	wg       sync.WaitGroup
	initOnce sync.Once
	stopOnce sync.Once

	connMap     map[*xtcp.Conn]*ServerConn
	connMapLock sync.RWMutex
	nextConnID  ConnID

	opt     *xtcp.Options
	xtcpsrv *xtcp.Server
}

// Serverが管理しているコネクションの状態を管理と、イベントハンドラの呼び出しを行う。
type ServerConn struct {
	ID           ConnID
	Conn         *xtcp.Conn
	Handler      *ConnHandler
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
		s.connMap = map[*xtcp.Conn]*ServerConn{}

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
	sc := s.getServerConn(conn)
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
			clientHello, ok := p.(*ClientHelloPacket)
			if !ok {
				s.error(fmt.Errorf("negotiation failed: client sends an unexpected packet: %#v", p))
				s.Stop(xtcp.StopImmediately)
				return
			}
			if !isCompatibleVersion(clientHello.ProtocolVersion) {
				// 対応していないバージョンなら、切断する。
				s.error(fmt.Errorf("negotiation failed: client version is not compatible"))
				s.Stop(xtcp.StopImmediately)
				return
			}

			srvHello := &ServerHelloPacket{
				ProtocolVersion: ProtocolVersion,
			}
			err := s.Send(srvHello)
			if err != nil {
				s.error(err)
				s.Stop(xtcp.StopImmediately)
				return
			}

			s.isNegotiated = true
			if s.Handler.Connected != nil {
				s.Handler.Connected(clientHello)
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
				s.Stop(xtcp.StopImmediately)
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
func (s *ServerConn) Send(p xtcp.Packet) error {
	if s.sendHandler == nil {
		return s.Conn.Send(p)
	} else {
		// call the mock method.
		s.sendHandler(s.Conn, p)
		return nil
	}
}

func (s *ServerConn) Stop(mode xtcp.StopMode) {
	if s.stopHandler == nil {
		s.Conn.Stop(mode)
	} else {
		// call the mock method.
		s.stopHandler(s.Conn, mode)
		// "xtcp.EventClosed" event occurs after calling the conn.Stop().
		// But the event is not occur when calling s.stopHandler().
		// So we occurs the event here.
		s.OnEvent(xtcp.EventClosed, s.Conn, nil)
	}
}

func (s *ServerConn) error(err error) {
	if s.Handler.Error != nil {
		s.Handler.Error(err)
	} else {
		log.Println("ERROR: ", err)
	}
}

func (s *Server) getServerConn(conn *xtcp.Conn) *ServerConn {
	s.connMapLock.Lock()
	defer s.connMapLock.Unlock()
	srvConn, ok := s.connMap[conn]
	if ok {
		return srvConn
	}

	srvConn = &ServerConn{
		ID:   s.nextConnID,
		Conn: conn,
	}
	srvConn.Handler = s.NewHandler(s.nextConnID, srvConn)
	s.connMap[conn] = srvConn
	s.nextConnID++
	return srvConn
}
