package protocol

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/yuuki0xff/xtcp"
)

const (
	DefaultMaxRetries = 10
	MinWaitTime       = 10 * time.Millisecond
	// 送信バッファに溜まっているパケットを強制的に排出する間隔。
	DefaultRefreshInterval = 100 * time.Millisecond
	// 送信バッファのサイズ。
	// 送信されるパケットサイズはこれよりも少し大きくなることがある。
	DefaultSendBufferSize = 1 << 20 // 1MiB

	packetChannelBufferSize = 1000000
)

var (
	InvalidProtocolError = errors.New("invalid protocol")
)

// クライアントで発生したイベントのイベントハンドラ。
// 不要なフィールドはnilにすることが可能。
type ClientHandler struct {
	Connected    func()
	Disconnected func()

	Error func(error)

	StartTrace func(*StartTraceCmdPacket)
	StopTrace  func(*StopTraceCmdPacket)
}

// ログサーバとの通信を行うクライアントの実装。
// 再接続機能が無いので、この実装を使用する側で適宜再接続を行うこと。
type Client struct {
	// Addr is "tcp://host:port"
	// 現在は、"unix:///path/to/socket/file" 形式のアドレスはサポートしていない。
	Addr    string
	Handler ClientHandler

	AppName      string
	Secret       string
	PingInterval time.Duration
	MaxRetries   int

	// 送信バッファに溜まっているパケットを強制的に送信する間隔。
	RefreshInterval time.Duration
	// 送信バッファのサイズ。
	// このサイズを超えた場合、バッファに溜まっているパケットをまとめて送信する。
	// 送信されるパケットのサイズは、BufferSizeよりも大きいことに留意すること。
	BufferSize int

	initOnce     sync.Once
	closeOnce    sync.Once
	negotiatedCh chan interface{}
	cancel       context.CancelFunc
	workerCtx    context.Context
	workerWg     sync.WaitGroup

	proto Proto
	// 送信バッファに入るのを待機しているパケット。
	// 突発的に多量のログが生成される状況でのパフォーマンス改善のため、バッファサイズは大きくしている。
	pktCh chan xtcp.Packet
	// MergePacketのpool。
	// 送信バッファのメモリ確保の回数削減のために使用している。
	mergePktPool sync.Pool

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
		if c.RefreshInterval == 0 {
			c.RefreshInterval = DefaultRefreshInterval
		}
		if c.BufferSize == 0 {
			c.BufferSize = DefaultSendBufferSize
		}
		c.pktCh = make(chan xtcp.Packet, packetChannelBufferSize)
	})
}

// Serve connects to the server and serve.
// this method is block until disconnected.
func (c *Client) Serve() error {
	c.Init()

	var addr string
	switch {
	case strings.HasPrefix(c.Addr, "unix://"):
		// TODO: support unix domain socket
		return InvalidProtocolError
	case strings.HasPrefix(c.Addr, "tcp://"):
		addr = strings.TrimPrefix(c.Addr, "tcp://")
	default:
		return InvalidProtocolError
	}

	// xtcpのバッファサイズが足りないと、パケットの送信に失敗していまう。
	// この問題を防ぐために、バッファサイズの最大サイズは十分に大きくしておく。
	c.opt = xtcp.NewOpts(c, &c.proto)
	c.opt.SetRecvBufInitSize(c.BufferSize * 2)
	c.opt.SetSendBufInitSize(c.BufferSize * 2)
	c.opt.SetRecvBufMaxSize(c.BufferSize * 10)
	c.opt.SetSendBufMaxSize(c.BufferSize * 10)
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

func (c *Client) Send(pkt xtcp.Packet) error {
	c.pktCh <- pkt
	return nil
}

func (c *Client) Close() error {
	var err error
	c.closeOnce.Do(func() {
		// send a shutdown message
		if err = c.Send(&ShutdownPacket{}); err != nil {
			err = errors.Wrap(err, "WARN: Client: can not send ShutdownPacket")
		}
		// request to worker shutdown
		c.cancel()

		// wait for worker ended before close TCP connection
		c.workerWg.Wait()

		c.xtcpconn.Stop(xtcp.StopGracefullyAndWait)
	})
	return err
}

func (c *Client) pingWorker() {
	defer c.workerWg.Done()

	ticker := time.NewTicker(c.PingInterval)
	for {
		select {
		case <-ticker.C:
			if err := c.Send(&PingPacket{}); err != nil {
				// TODO: try to reconnect
				panic(err)
			}

		case <-c.workerCtx.Done():
			return
		}
	}
}

// pktChから送られてくるshort packetを統合して、large packetにしてから送信する。
// RefreshInterval間隔で、強制的にバッファの中身を送信する。
func (c *Client) mergeWorker() {
	defer c.workerWg.Done()

	c.mergePktPool = sync.Pool{
		New: func() interface{} {
			return &MergePacket{
				Proto: &c.proto,
			}
		},
	}
	mergePkt := c.mergePktPool.Get().(*MergePacket)

	ticker := time.NewTicker(c.RefreshInterval)
	for {
		select {
		case pkt := <-c.pktCh:
			// TODO: mergePkt.Merge(pkt)は時間がかかる処理である。マルチスレッド化する。
			mergePkt.Merge(pkt)
			if mergePkt.Len() >= c.BufferSize {
				c.xtcpconn.Send(mergePkt)
				mergePkt = c.mergePktPool.Get().(*MergePacket)
				mergePkt.Reset()
			}
		case <-ticker.C:
			if mergePkt.Len() > 0 {
				c.xtcpconn.Send(mergePkt)
				mergePkt = c.mergePktPool.Get().(*MergePacket)
				mergePkt.Reset()
			}
		case <-c.workerCtx.Done():
			ticker.Stop()
			c.xtcpconn.Send(mergePkt)
			return
		}
	}
}

// p will be nil when event is EventAccept/EventConnected/EventClosed
func (c *Client) OnEvent(et xtcp.EventType, conn *xtcp.Conn, p xtcp.Packet) {
	switch et {
	case xtcp.EventConnected:
		// send client header packet
		pkt := &ClientHelloPacket{
			AppName:         c.AppName,
			ClientSecret:    c.Secret,
			ProtocolVersion: ProtocolVersion,
		}
		if err := c.xtcpconn.Send(pkt); err != nil {
			// TODO: try to reconnect
			panic(err)
		}
	case xtcp.EventRecv:
		// if first time, a packet MUST BE ServerHelloPacket type.
		if !c.isNegotiated {
			pkt, ok := p.(*ServerHelloPacket)
			if !ok {
				c.xtcpconn.Stop(xtcp.StopImmediately)
				return
			}
			if !isCompatibleVersion(pkt.ProtocolVersion) {
				// 対応していないバージョンなら、切断する。
				conn.Stop(xtcp.StopImmediately)
				return
			}

			c.workerWg.Add(2)
			go c.pingWorker()
			go c.mergeWorker()

			c.negotiated()

			if c.Handler.Connected != nil {
				c.Handler.Connected()
			}
		} else {
			switch pkt := p.(type) {
			case *PingPacket:
				// do nothing
			case *ShutdownPacket:
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
				conn.Stop(xtcp.StopImmediately)
			case *RawFuncLogPacket:
				conn.Stop(xtcp.StopImmediately)
			default:
				panic(fmt.Sprintf("BUG: Client: Client receives a invalid Packet: %+v %+v", pkt, reflect.TypeOf(pkt)))
			}
		}
	case xtcp.EventSend:
		if _, ok := p.(*MergePacket); ok {
			// 送信が完了したMergePacketは、poolに追加して再利用する。
			c.mergePktPool.Put(p)
		}
	case xtcp.EventClosed:

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
