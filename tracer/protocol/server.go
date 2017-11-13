package protocol

import (
	"net"
	"strings"

	"time"

	"sync"

	"log"

	"fmt"

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
	wg       sync.WaitGroup
	stopOnce sync.Once

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
	s.wg.Add(1)
	defer s.wg.Done()

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
				s.Handler.Connected()
			}
		} else {
			switch pkt := p.(type) {
			case PingPacket:
				// do nothing
			case ShutdownPacket:
				// TODO: dummy code
				pkt.String()
				log.Println("INFO: Server: get a shutdown msg")
				conn.Stop(xtcp.StopImmediately)
				return
			case StartTraceCmdPacket:
				log.Println("ERROR: Server: invalid packet: StartTraceCmdPacket is not allowed")
				conn.Stop(xtcp.StopImmediately)
				return
			case StopTraceCmdPacket:
				log.Println("ERROR: Server: invalid packet: StopTraceCmdPacket is not allowed")
				conn.Stop(xtcp.StopImmediately)
				return
			case SymbolPacket:
				// TODO
			case RawFuncLogNewPacket:
				// TODO
			default:
				panic(fmt.Sprintf("BUG: Server: Server receives a invalid Packet: %+v", pkt))
			}
		}
	case xtcp.EventSend:
	case xtcp.EventClosed:
		log.Println("INFO: Server: disconnected")
		if s.Handler.Disconnected != nil {
			s.Handler.Disconnected()
		}
	}
}
