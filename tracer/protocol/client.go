package protocol

import (
	"context"
	"errors"
	"strings"

	"log"
	"sync"
	"time"

	"github.com/xfxdev/xtcp"
)

var (
	InvalidProtocolError = errors.New("invalid protocol")
)

type ClientHandler struct {
	Connected    func()
	Disconnected func()

	Error func(error)

	StartTrace func(*StartTraceCmdArgs)
	StopTrace  func(*StopTraceCmdArgs)
}

type Client struct {
	// Addr is "tcp://host:port"
	// 現在は、"unix:///path/to/socket/file" 形式のアドレスはサポートしていない。
	Addr    string
	Handler ClientHandler

	AppName      string
	Secret       string
	PingInterval time.Duration

	cancel    context.CancelFunc
	workerCtx context.Context
	workerWg  sync.WaitGroup

	opt          *xtcp.Options
	xtcpconn     *xtcp.Conn
	isNegotiated bool
}

func (c *Client) init() {
	c.workerCtx, c.cancel = context.WithCancel(context.Background())
	if c.PingInterval == time.Duration(0) {
		c.PingInterval = DefaultPingInterval
	}
}

func (c *Client) Connect() error {
	log.Println("INFO: clinet: connected")
	c.init()

	var addr string
	switch {
	case strings.HasPrefix(c.Addr, "unix://"):
		// TODO
		return InvalidProtocolError
	case strings.HasPrefix(c.Addr, "tcp://"):
		addr = strings.TrimPrefix(c.Addr, "tcp://")
	default:
		return InvalidProtocolError
	}

	prt := &Proto{}
	c.opt = xtcp.NewOpts(c, prt)
	c.xtcpconn = xtcp.NewConn(c.opt)
	return c.xtcpconn.DialAndServe(addr)
}

func (c *Client) Send(msgType MessageType, data xtcp.Packet) {
	log.Printf("DEBUG: client: send message type=%+v, data=%+v\n", msgType, data)
	c.xtcpconn.Send(&MessageHeader{
		MessageType: msgType,
		Messages:    1,
	})
	c.xtcpconn.Send(data)
}

func (c *Client) Close() error {
	log.Println("INFO: client: closing a connection")
	defer log.Println("DEBUG: client: closed a connection")
	if c.cancel != nil {
		// send a shutdown message
		c.Send(ShutdownMsg, &ShutdownPacket{})

		// request to worker shutdown
		c.cancel()
		c.cancel = nil

		// wait for worker ended before close TCP connection
		c.workerWg.Wait()
	}
	return nil
}

func (c *Client) pingWorker() {
	log.Println("DEBUG: client: start ping worker")
	defer log.Println("DEBUG: client: stop ping worker")
	defer c.workerWg.Done()

	timer := time.NewTicker(c.PingInterval)

	for {
		select {
		case <-timer.C:
			log.Println("DEBUG: client: send ping message")
			c.Send(PingMsg, &PingPacket{})

		case <-c.workerCtx.Done():
			return
		}
	}
}

// p will be nil when event is EventAccept/EventConnected/EventClosed
func (c *Client) OnEvent(et xtcp.EventType, conn *xtcp.Conn, p xtcp.Packet) {
	switch et {
	case xtcp.EventConnected:
		// send client header packet
		pkt := &ClientHeader{
			AppName:         c.AppName,
			ClientSecret:    c.Secret,
			ProtocolVersion: ProtocolVersion,
		}
		c.xtcpconn.Send(pkt)
	case xtcp.EventRecv:
		// 初めてのパケットを受け取ったときには、サーバハンドラとしてデコードする
		// if first time, a packet MUST BE ServerHeader type.
		if !c.isNegotiated {
			pkt, ok := p.(ServerHeader)
			if !ok {
				log.Printf("ERROR: invalid server header")
				c.xtcpconn.Stop(xtcp.StopImmediately)
				return
			}
			if isCompatibleVersion(pkt.ProtocolVersion) {
				// 対応していないバージョンなら、切断する。
				conn.Stop(xtcp.StopImmediately)
				return
			}

			c.workerWg.Add(1)
			go c.pingWorker()

			c.isNegotiated = true

			if c.Handler.Connected != nil {
				c.Handler.Connected()
			}
		} else {
			switch pkt := p.(type) {
			case PingPacket:
				// do nothing
			case ShutdownPacket:
				// TODO: dummy code
				pkt.String()
				log.Println("INFO: client: get a shutdown msg")
				conn.Stop(xtcp.StopImmediately)
				return
			case StartTraceCmdPacket:
				// TODO
			case StopTraceCmdPacket:
				// TODO
			case SymbolPacket:
				log.Println("ERROR: invalid packet: SymbolPacket is not allowed")
				conn.Stop(xtcp.StopImmediately)
			case RawFuncLogNewPacket:
				log.Println("ERROR: invalid packet: RawFuncLogNewPacket is not allowed")
				conn.Stop(xtcp.StopImmediately)
			default:
				panic("bug")
			}
		}
	case xtcp.EventSend:
	case xtcp.EventClosed:
		if c.Handler.Disconnected != nil {
			c.Handler.Disconnected()
		}
	}
}
