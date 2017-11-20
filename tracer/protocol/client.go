package protocol

import (
	"context"
	"errors"
	"strings"

	"log"
	"sync"
	"time"

	"fmt"

	"reflect"

	"github.com/yuuki0xff/xtcp"
)

const (
	DefaultMaxRetries = 10
	MinWaitTime       = 10 * time.Millisecond
)

var (
	InvalidProtocolError = errors.New("invalid protocol")
)

type ClientHandler struct {
	Connected    func()
	Disconnected func()

	Error func(error)

	StartTrace func(*StartTraceCmdPacket)
	StopTrace  func(*StopTraceCmdPacket)
}

type Client struct {
	// Addr is "tcp://host:port"
	// 現在は、"unix:///path/to/socket/file" 形式のアドレスはサポートしていない。
	Addr    string
	Handler ClientHandler

	AppName      string
	Secret       string
	PingInterval time.Duration
	MaxRetries   int

	initOnce     sync.Once
	closeOnce    sync.Once
	negotiatedCh chan interface{}
	cancel       context.CancelFunc
	workerCtx    context.Context
	workerWg     sync.WaitGroup

	opt          *xtcp.Options
	xtcpconn     *xtcp.Conn
	isNegotiated bool
}

func (c *Client) Init() {
	c.initOnce.Do(func() {
		if c.MaxRetries == 0 {
			c.MaxRetries = DefaultMaxRetries
		}

		c.negotiatedCh = make(chan interface{})

		c.workerCtx, c.cancel = context.WithCancel(context.Background())
		if c.PingInterval == time.Duration(0) {
			c.PingInterval = DefaultPingInterval
		}
	})
}

// Serve connects to the server and serve.
// this method is block until disconnected.
func (c *Client) Serve() error {
	log.Println("INFO: Client: connected")
	c.Init()

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

	// retry loop
	retries := 0
	waitTime := MinWaitTime
	for {
		err := c.xtcpconn.DialAndServe(addr)
		if err != nil {
			// occurs error when dialing
			if retries < c.MaxRetries {
				retries++
				time.Sleep(waitTime)
				waitTime *= 2
				continue
			} else {
				return err
			}
		}
		return nil
	}
}

func (c *Client) Send(data xtcp.Packet) error {
	log.Printf("DEBUG: Client: send packet: %+v\n", data)
	return c.xtcpconn.Send(data)
}

func (c *Client) Close() error {
	log.Println("INFO: Client: closing a connection")
	defer log.Println("DEBUG: Client: closed a connection")
	c.closeOnce.Do(func() {
		// send a shutdown message
		if err := c.Send(&ShutdownPacket{}); err != nil {
			log.Printf("WARN: Client: can not send ShutdownPacket")
		}
		// request to worker shutdown
		c.cancel()

		// wait for worker ended before close TCP connection
		log.Println("DEBUG: Client: wait for worker ended")
		c.workerWg.Wait()

		log.Println("DEBUG: Client: closing a connection")
		c.xtcpconn.Stop(xtcp.StopGracefullyAndWait)
		log.Println("DEBUG: Client: closed a connection")
	})
	return nil
}

func (c *Client) pingWorker() {
	log.Println("DEBUG: Client: start ping worker")
	defer log.Println("DEBUG: Client: stop ping worker")
	defer c.workerWg.Done()

	timer := time.NewTicker(c.PingInterval)

	for {
		select {
		case <-timer.C:
			log.Println("DEBUG: Client: send ping message")
			if err := c.Send(&PingPacket{}); err != nil {
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
			// TODO: try to reconnect
			panic(err)
		}
	case xtcp.EventRecv:
		// if first time, a packet MUST BE ServerHelloPacket type.
		if !c.isNegotiated {
			pkt, ok := p.(*ServerHelloPacket)
			if !ok {
				log.Printf("ERROR: Client: invalid ServerHelloPacket")
				c.xtcpconn.Stop(xtcp.StopImmediately)
				return
			}
			log.Printf("DEBUG: Client: received a ServerHelloPacket: %+v", pkt)
			log.Printf("DEBUG: Client: ProtocolVersion server=%s client=%s", pkt.ProtocolVersion, ProtocolVersion)
			if !isCompatibleVersion(pkt.ProtocolVersion) {
				// 対応していないバージョンなら、切断する。
				log.Printf("ERROR: Client: mismatch the protocol version: server=%s client=%s", pkt.ProtocolVersion, ProtocolVersion)
				conn.Stop(xtcp.StopImmediately)
				return
			}
			log.Println("DEBUG: Client: success negotiation process")

			c.workerWg.Add(1)
			go c.pingWorker()

			c.negotiated()

			if c.Handler.Connected != nil {
				c.Handler.Connected()
			}
		} else {
			log.Printf("DEBUG: Client: recieved a packet: %+v", p)
			switch pkt := p.(type) {
			case *PingPacket:
				// do nothing
			case *ShutdownPacket:
				log.Println("INFO: Client: get a shutdown msg")
				conn.Stop(xtcp.StopImmediately)
				return
			case *StartTraceCmdPacket:
				if c.Handler.StartTrace != nil {
					c.Handler.StartTrace(pkt)
				}
			case *StopTraceCmdPacket:
				if c.Handler.StopTrace != nil {
					c.Handler.StopTrace(pkt)
				}
			case *SymbolPacket:
				log.Println("ERROR: Client: invalid packet: SymbolPacket is not allowed")
				conn.Stop(xtcp.StopImmediately)
			case *RawFuncLogNewPacket:
				log.Println("ERROR: Client: invalid packet: RawFuncLogNewPacket is not allowed")
				conn.Stop(xtcp.StopImmediately)
			default:
				panic(fmt.Sprintf("BUG: Client: Client receives a invalid Packet: %+v %+v", pkt, reflect.TypeOf(pkt)))
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

// WaitNegotiation wait for negotiation to be finish
func (c *Client) WaitNegotiation() {
	<-c.negotiatedCh
}

func (c *Client) negotiated() {
	c.isNegotiated = true
	close(c.negotiatedCh)
}
