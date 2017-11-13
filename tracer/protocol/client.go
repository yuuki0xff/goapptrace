package protocol

import (
	"context"
	"errors"
	"strings"

	"log"
	"sync"
	"time"

	"fmt"

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

func (c *Client) Send(msgType MessageType, data xtcp.Packet) error {
	log.Printf("DEBUG: client: send message type=%+v, data=%+v\n", msgType, data)
	err := c.xtcpconn.Send(&MessageHeader{
		MessageType: msgType,
		Messages:    1,
	})
	if err != nil {
		return err
	}
	return c.xtcpconn.Send(data)
}

func (c *Client) Close() error {
	log.Println("INFO: client: closing a connection")
	defer log.Println("DEBUG: client: closed a connection")
	if c.cancel != nil {
		// send a shutdown message
		if err := c.Send(ShutdownMsg, &ShutdownPacket{}); err != nil {
			log.Printf("WARN: client: can not send ShutdownPacket")
		}
		// request to worker shutdown
		c.cancel()
		c.cancel = nil

		// wait for worker ended before close TCP connection
		log.Println("DEBUG: client: wait for worker ended")
		c.workerWg.Wait()

		log.Println("DEBUG: client: closing a connection")
		c.xtcpconn.Stop(xtcp.StopGracefullyAndWait)
		log.Println("DEBUG: client: closed a connection")
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
			if err := c.Send(PingMsg, &PingPacket{}); err != nil {
				// TODO: try to reconnect
				panic(err)
			}

		case <-c.workerCtx.Done():
			return
		}
	}
}

// p will be nil when event is EventAccept/EventConnected/EventClosed
func (c *Client) OnEvent(et xtcp.EventType, conn *xtcp.Conn, p xtcp.Packet) {
	switch et {
	case xtcp.EventConnected:
		log.Println("DEBUG: Client: connected to the server")
		// send client header packet
		pkt := &ClientHelloPacket{
			AppName:         c.AppName,
			ClientSecret:    c.Secret,
			ProtocolVersion: ProtocolVersion,
		}
		log.Printf("DEBUG: Client: send a ClientHelloPacket: %+v", pkt)
		if err := c.xtcpconn.Send(pkt); err != nil {
			// try to reconnect
			panic(err)
		}
	case xtcp.EventRecv:
		// 初めてのパケットを受け取ったときには、サーバハンドラとしてデコードする
		// if first time, a packet MUST BE ServerHelloPacket type.
		if !c.isNegotiated {
			pkt, ok := p.(ServerHelloPacket)
			if !ok {
				log.Printf("ERROR: invalid server header")
				c.xtcpconn.Stop(xtcp.StopImmediately)
				return
			}
			log.Printf("DEBUG: Client: received a ServerHelloPacket: %+v", pkt)
			if isCompatibleVersion(pkt.ProtocolVersion) {
				// 対応していないバージョンなら、切断する。
				conn.Stop(xtcp.StopImmediately)
				return
			}
			log.Println("DEBUG: Client: success negotiation process")

			c.workerWg.Add(1)
			go c.pingWorker()

			c.isNegotiated = true

			if c.Handler.Connected != nil {
				c.Handler.Connected()
			}
		} else {
			log.Printf("DEBUG: Client: recieved a packet: %+v", p)
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
				panic(fmt.Sprintf("bug: Client receives a invalid Packet: %+v", pkt))
			}
		}
	case xtcp.EventSend:
	case xtcp.EventClosed:
		log.Println("DEBUG: Client: connection closed")

		// request worker shutdown
		c.cancel()

		if c.Handler.Disconnected != nil {
			c.Handler.Disconnected()
		}
	}
}
