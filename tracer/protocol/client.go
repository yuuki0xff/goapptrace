package protocol

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/yuuki0xff/xtcp"
)

const (
	DefaultMaxRetries = 10
	MinWaitTime       = 10 * time.Millisecond

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

	Shutdown   func(*ShutdownPacket)
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

	PID          uint64
	AppName      string
	Host         string
	Secret       string
	PingInterval time.Duration
	MaxRetries   int
	BufferOpt    BufferOption

	initOnce     sync.Once
	closeOnce    sync.Once
	negotiatedCh chan interface{}
	cancel       context.CancelFunc
	workerCtx    context.Context
	workerWg     sync.WaitGroup

	mergeSender mergeSender
	proto       Proto

	opt          *xtcp.Options
	xtcpconn     *xtcp.Conn
	isNegotiated bool
}

// short packetを統合して、large packetにしてから送信する。
// packet送信によるパフォーマンス低下を軽減するために利用する。
type mergeSender struct {
	Conn  *xtcp.Conn
	Proto *Proto
	Opt   BufferOption

	// 送信バッファに入るのを待機しているパケット。
	// 突発的に多量のログが生成される状況でのパフォーマンス改善のため、バッファサイズは大きくしている。
	// 中身は全てMergePacketである。
	ch chan *MergePacket
	// MergePacketのpool。
	// 送信バッファのメモリ確保の回数削減のために使用する。
	pool sync.Pool

	m sync.Mutex
	// 構築中のmergePacket。
	// アクセスする場合はlockを取ってからアクセスすること。
	mergePkt *MergePacket

	// RefreshWorker の ctx が中断されたら、0以外の値になる。
	// この値が0以外のときにパケットを送信すると、panicする。
	stopped int64
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
		c.BufferOpt.SetDefault()
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
	c.BufferOpt.Xtcp.Set(c.opt)
	c.xtcpconn = xtcp.NewConn(c.opt)

	// initialize mergeSender
	c.mergeSender = mergeSender{
		Conn:  c.xtcpconn,
		Proto: &c.proto,
		Opt:   c.BufferOpt,
	}
	c.mergeSender.Init()

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

// Send sends a packet asynchronously.
// pkt is marshalled immediately. Caller can be reuse pkt after return this function.
func (c *Client) Send(pkt xtcp.Packet) error {
	return c.mergeSender.Send(pkt)
}

// SendLarge sends a large packet.
func (c *Client) SendLarge(largePkt xtcp.Packet) error {
	return c.mergeSender.SendLarge(largePkt)
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
			ticker.Stop()
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
			PID:             c.PID,
			AppName:         c.AppName,
			Host:            c.Host,
			ClientSecret:    c.Secret,
			ProtocolVersion: ProtocolVersion,
		}
		if err := c.xtcpconn.Send(pkt); err != nil {
			c.error(err)
			c.stop(xtcp.StopImmediately)
			return
		}
	case xtcp.EventRecv:
		// if first time, a packet MUST BE ServerHelloPacket type.
		if !c.isNegotiated {
			pkt, ok := p.(*ServerHelloPacket)
			if !ok {
				c.error(fmt.Errorf("negotiation failed: server sends an unexpected packet: %#v", p))
				c.stop(xtcp.StopImmediately)
				return
			}
			if !isCompatibleVersion(pkt.ProtocolVersion) {
				// 対応していないバージョンなら、切断する。
				c.error(fmt.Errorf("negotiation failed: server version is not compatible"))
				c.stop(xtcp.StopImmediately)
				return
			}

			c.workerWg.Add(2)
			go c.pingWorker()
			go c.mergeSender.RefreshWorker(&c.workerWg, c.workerCtx)

			c.negotiated()

			if c.Handler.Connected != nil {
				c.Handler.Connected()
			}
		} else {
			switch pkt := p.(type) {
			case *PingPacket:
				// do nothing
			case *ShutdownPacket:
				if c.Handler.Shutdown != nil {
					c.Handler.Shutdown(pkt)
				} else {
					defer log.Fatal("killed by goapptrace")
				}
				c.cancel()
				c.stop(xtcp.StopGracefullyAndWait)
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
				c.error(fmt.Errorf("client receives an unexpected packet: %#v", pkt))
				c.stop(xtcp.StopImmediately)
				return
			}
		}
	case xtcp.EventSend:
		if _, ok := p.(*MergePacket); ok {
			// 送信が完了したMergePacketは、poolに追加して再利用する。
			c.mergeSender.Put(p)
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

func (c *Client) error(err error) {
	if c.Handler.Error != nil {
		c.Handler.Error(err)
	} else {
		log.Println("ERROR:", err)
	}
}
func (c *Client) stop(mode xtcp.StopMode) {
	c.xtcpconn.Stop(mode)
}

func (ms *mergeSender) Init() {
	ms.ch = make(chan *MergePacket, packetChannelBufferSize)
	ms.pool = sync.Pool{
		New: func() interface{} {
			return &MergePacket{
				Proto: ms.Proto,
				// MergePacketのサイズは、BufferSizeよりも大きくなる。
				// そのため、少し大きめのバッファを確保しておく。
				BufferSize: ms.Opt.MaxSmallPacketSize + 2048,
			}
		},
	}
}

// いくつかのパケットをまとめて、一括送信する。
// pkt is marshalled immediately. Caller can be reuse pkt after return this function.
func (ms *mergeSender) Send(pkt xtcp.Packet) error {
	ms.m.Lock()
	defer ms.m.Unlock()
	return ms.sendNolock(pkt)
}
func (ms *mergeSender) sendNolock(pkt xtcp.Packet) error {
	if atomic.LoadInt64(&ms.stopped) != 0 {
		panic("ms.stopped != 0")
	}
	if ms.mergePkt == nil {
		ms.mergePkt = ms.pool.Get().(*MergePacket)
		ms.mergePkt.Reset()
	}
	mp := ms.mergePkt

	mp.Merge(pkt)
	if mp.Len() >= ms.Opt.MaxSmallPacketSize {
		ms.mergePkt = nil
		return ms.Conn.Send(mp)
	}
	return nil
}

func (ms *mergeSender) SendLarge(largePkt xtcp.Packet) error {
	ms.m.Lock()
	defer ms.m.Unlock()
	if atomic.LoadInt64(&ms.stopped) != 0 {
		panic("ms.stopped != 0")
	}
	if err := ms.refreshNolock(); err != nil {
		return err
	}

	return ms.Conn.Send(marshalLargePacket(ms.Proto, largePkt))
}

// 送信が完了したMergePacketをpoolに追加して、MergePacketを再利用する。
func (ms *mergeSender) Put(pkt xtcp.Packet) {
	ms.pool.Put(pkt)
}

// 強制的にバッファの中身を送信する。
func (ms *mergeSender) Refresh() error {
	ms.m.Lock()
	defer ms.m.Unlock()
	return ms.refreshNolock()
}
func (ms *mergeSender) refreshNolock() error {
	if atomic.LoadInt64(&ms.stopped) != 0 {
		panic("ms.stopped != 0")
	}
	pkt := ms.mergePkt
	if pkt == nil || pkt.Len() == 0 {
		return nil
	}
	ms.mergePkt = nil
	return ms.Conn.Send(pkt)
}

// mergeSender を止めるときに呼び出す。
// mergeSender.stopped フラグを立てて、これ以降はパケットの送信が出来ないように設定する。
func (ms *mergeSender) refreshLast() error {
	atomic.StoreInt64(&ms.stopped, 1)
	ms.m.Lock()
	defer ms.m.Unlock()
	pkt := ms.mergePkt
	if pkt == nil || pkt.Len() == 0 {
		return nil
	}
	ms.mergePkt = nil
	return ms.Conn.Send(pkt)
}

// RefreshInterval間隔で、強制的にバッファの中身を送信する。
// また、ctxが終了したときにもバッファの中身を送信する。
// バッファに滞留してしまい、いつまでもサーバにパケットが届かなくなる問題を防ぐ。
// 送信中にエラーが発生した場合、panicする。
func (ms *mergeSender) RefreshWorker(wg *sync.WaitGroup, ctx context.Context) {
	defer wg.Done()

	ticker := time.NewTicker(ms.Opt.RefreshInterval)
	for {
		select {
		case <-ticker.C:
			err := ms.Refresh()
			if err != nil {
				// TODO: imrpove error handling
				log.Panicln(err)
			}
		case <-ctx.Done():
			ticker.Stop()
			err := ms.refreshLast()
			if err != nil {
				// TODO: imrpove error handling
				log.Panicln(err)
			}
			return
		}
	}
}
