package protocol

import (
	"net"
	"strings"

	"time"

	"sync"

	"log"

	"github.com/xfxdev/xtcp"
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
)

type ServerHandler struct {
	Connected    func()
	Disconnected func()

	Error func(error)

	Symbols    func(*logutil.Symbols)
	RawFuncLog func(*logutil.RawFuncLogNew)
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
	workerWg sync.WaitGroup

	opt          *xtcp.Options
	xtcpsrv      *xtcp.Server
	isNegotiated bool
}

func (s *Server) Listen() error {
	var addr string
	var err error

	switch {
	case strings.HasPrefix(s.Addr, "unix://"):
		// TODO
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

	if s.MaxBufferedMsgs <= 0 {
		s.MaxBufferedMsgs = DefaultMaxBufferedMsgs
	}
	if s.PingInterval == time.Duration(0) {
		s.PingInterval = DefaultPingInterval
	}
	return nil
}

func (s *Server) Serve() {
	prt := &Proto{}
	s.opt = xtcp.NewOpts(s, prt)
	s.xtcpsrv = xtcp.NewServer(s.opt)
	s.xtcpsrv.Serve(s.listener)
}

func (s *Server) ActualAddr() string {
	addr := s.listener.Addr()
	return addr.Network() + "://" + addr.String()
}

func (s *Server) Close() error {
	log.Println("INFO: server: closeing a connection")
	defer log.Println("DEBUG: server: closing a connection ... done")

	s.xtcpsrv.Stop(xtcp.StopImmediately)
	return nil
}

func (s *Server) Wait() {
	s.workerWg.Wait()
}

// p will be nil when event is EventAccept/EventConnected/EventClosed
func (s *Server) OnEvent(et xtcp.EventType, conn *xtcp.Conn, p xtcp.Packet) {
	switch et {
	case xtcp.EventAccept:
		// wait for client header packet to be received.
	case xtcp.EventRecv:
		if !s.isNegotiated {
			// check client header.
			pkt, ok := p.(ClientHeader)
			if !ok {
				log.Printf("ERROR: invalid client header")
				conn.Stop(xtcp.StopImmediately)
				return
			}
			if isCompatibleVersion(pkt.ProtocolVersion) {
				conn.Stop(xtcp.StopImmediately)
				return
			}
			conn.Send(&ServerHeader{
				ProtocolVersion: ProtocolVersion,
			})
			s.isNegotiated = true

			if s.Handler.Connected != nil {
				s.Handler.Connected()
			}
		} else {
			switch pkt := p.(type) {
			case PingPacket:
				// do nothing
			case ShutdownPacket:
				// TODO: dummy code
				pkt.String()
				log.Println("INFO: server: get a shutdown msg")
				conn.Stop(xtcp.StopImmediately)
				return
			case StartTraceCmdPacket:
				log.Println("ERROR: invalid packet: StartTraceCmdPacket is not allowed")
				conn.Stop(xtcp.StopImmediately)
				return
			case StopTraceCmdPacket:
				log.Println("ERROR: invalid packet: StopTraceCmdPacket is not allowed")
				conn.Stop(xtcp.StopImmediately)
				return
			case SymbolPacket:
				// TODO
			case RawFuncLogNewPacket:
				// TODO
			}
		}
	case xtcp.EventSend:
	case xtcp.EventClosed:
		if s.Handler.Disconnected != nil {
			s.Handler.Disconnected()
		}
	}
}
