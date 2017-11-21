package protocol

import (
	"net"
	"strings"

	"time"

	"sync"

	"log"

	"fmt"

	"reflect"

	"github.com/yuuki0xff/goapptrace/tracer/logutil"
	"github.com/yuuki0xff/xtcp"
)

const (
	DefaultTCPPort = 8600
	MaxListenTries = 100
)

type ConnID int64

type ServerHandler struct {
	Connected    func(id ConnID)
	Disconnected func(id ConnID)

	Error func(id ConnID, err error)

	Symbols    func(id ConnID, symbols *logutil.Symbols)
	RawFuncLog func(id ConnID, funclog *logutil.RawFuncLogNew)
}

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

	opt          *xtcp.Options
	xtcpsrv      *xtcp.Server
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
	log.Println("INFO: Server: closeing a connection")
	defer log.Println("DEBUG: Server: closing a connection ... done")

	s.stopOnce.Do(func() {
		// Stop method MUST NOT be called many times.
		s.xtcpsrv.Stop(xtcp.StopGracefullyAndWait)
	})
	return nil
}

func (s *Server) Wait() {
	s.wg.Wait()
}

// p will be nil when event is EventAccept/EventConnected/EventClosed
func (s *Server) OnEvent(et xtcp.EventType, conn *xtcp.Conn, p xtcp.Packet) {
	switch et {
	case xtcp.EventAccept:
		log.Println("DEBUG: Server: accepted a connection. wait for receives a ClientHelloPacket")
		// wait for client header packet to be received.
	case xtcp.EventRecv:
		log.Printf("DEBUG: Server: received a Packet: %+v", p)
		if !s.isNegotiated {
			// check client header.
			pkt, ok := p.(*ClientHelloPacket)
			if !ok {
				log.Printf("ERROR: Server: invalid ClientHelloPacket")
				conn.Stop(xtcp.StopImmediately)
				return
			}
			log.Printf("DEBUG: Server: received a ClientHelloPacket: %+v", pkt)
			log.Printf("DEBUG: Server: ProtocolVersion server=%s client=%s", ProtocolVersion, pkt.ProtocolVersion)
			if !isCompatibleVersion(pkt.ProtocolVersion) {
				// 対応していないバージョンなら、切断する。
				log.Printf("ERROR: Server: mismatch the protocol version: server=%s client=%s", ProtocolVersion, pkt.ProtocolVersion)
				conn.Stop(xtcp.StopImmediately)
				return
			}

			packet := &ServerHelloPacket{
				ProtocolVersion: ProtocolVersion,
			}
			log.Printf("DEBUG: Server: send a ServerHelloPacket: %+v", packet)
			if err := conn.Send(packet); err != nil {
				// TODO: try to reconnect
				panic(err)
			}
			s.isNegotiated = true

			log.Println("DEBUG: Server: success negotiation process")
			if s.Handler.Connected != nil {
				s.Handler.Connected(s.getConnID(conn))
			}
		} else {
			switch pkt := p.(type) {
			case *PingPacket:
				// do nothing
			case *ShutdownPacket:
				log.Println("INFO: Server: get a shutdown msg")
				conn.Stop(xtcp.StopImmediately)
				return
			case *StartTraceCmdPacket:
				log.Println("ERROR: Server: invalid packet: StartTraceCmdPacket is not allowed")
				conn.Stop(xtcp.StopImmediately)
				return
			case *StopTraceCmdPacket:
				log.Println("ERROR: Server: invalid packet: StopTraceCmdPacket is not allowed")
				conn.Stop(xtcp.StopImmediately)
				return
			case *SymbolPacket:
				if s.Handler.Symbols != nil {
					s.Handler.Symbols(s.getConnID(conn), pkt.Symbols)
				}
			case *RawFuncLogNewPacket:
				if s.Handler.RawFuncLog != nil {
					s.Handler.RawFuncLog(s.getConnID(conn), pkt.FuncLog)
				}
			default:
				panic(fmt.Sprintf("BUG: Server: Server receives a invalid Packet: %+v %+v", pkt, reflect.TypeOf(pkt)))
			}
		}
	case xtcp.EventSend:
	case xtcp.EventClosed:
		log.Println("INFO: Server: disconnected")
		if s.Handler.Disconnected != nil {
			s.Handler.Disconnected(s.getConnID(conn))
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
