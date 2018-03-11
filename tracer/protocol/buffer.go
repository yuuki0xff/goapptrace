package protocol

import (
	"time"

	"github.com/yuuki0xff/xtcp"
)

const (
	// 送信バッファに溜まっているパケットを強制的に排出する間隔。
	DefaultRefreshInterval = 100 * time.Millisecond

	// 最も頻繁に送受信されるパケットの最大サイズ。
	// client側はこれらのパケットをバッファリングしており、非同期送信ができる。
	DefaultMaxSmallPacketSize = 1 << 20 //1 MiB
	// 送受信可能なパケットの最大サイズ。
	// このパケットはバッファリングされず、送信完了までブロックする。
	// パフォーマンス低下の原因になる。
	DefaultMaxLargePacketSize = 100 * 1 << 20 // 100MiB

	// xtcpのバッファのサイズ。
	// 送信/受信されるパケットサイズはこれよりも少し大きくなることがある。

	// 送信キューに滞留可能なパケット数
	DefaultSendListLen     = 30
	DefaultRecvBufInitSize = 1 << 20       // 1MiB
	DefaultRecvBufMaxSize  = 100 * 1 << 20 // 100MiB
	DefaultSendBufInitSize = 1 << 20       // 1MiB
	DefaultSendBufMaxSize  = 100 * 1 << 20 // 100MiB
)

type BufferOption struct {
	// 送信バッファに溜まっているパケットを強制的に送信する間隔。
	RefreshInterval time.Duration

	MaxSmallPacketSize int
	MaxLargePacketSize int

	// xtcp関連
	Xtcp XtcpBufferOption
}

func (opt *BufferOption) SetDefault() {
	if opt.RefreshInterval == 0 {
		opt.RefreshInterval = DefaultRefreshInterval
	}
	if opt.MaxSmallPacketSize == 0 {
		opt.MaxSmallPacketSize = DefaultMaxSmallPacketSize
	}
	if opt.MaxLargePacketSize == 0 {
		opt.MaxLargePacketSize = DefaultMaxLargePacketSize
	}
	opt.Xtcp.SetDefault()
}

type XtcpBufferOption struct {
	SendListLen     int
	RecvBufInitSize int
	SendBufInitSize int
	RecvBufMaxSize  int
	SendBufMaxSize  int
}

func (opt *XtcpBufferOption) SetDefault() {
	if opt.SendListLen == 0 {
		opt.SendListLen = DefaultSendListLen
	}
	if opt.RecvBufInitSize == 0 {
		opt.RecvBufInitSize = DefaultRecvBufInitSize
	}
	if opt.SendBufInitSize == 0 {
		opt.SendBufInitSize = DefaultSendBufInitSize
	}
	if opt.RecvBufInitSize == 0 {
		opt.RecvBufInitSize = DefaultRecvBufMaxSize
	}
	if opt.SendBufMaxSize == 0 {
		opt.SendBufMaxSize = DefaultSendBufMaxSize
	}
}

func (opt *XtcpBufferOption) Set(o *xtcp.Options) {
	o.SetSendListLen(opt.SendListLen)
	o.SetRecvBufInitSize(opt.RecvBufInitSize)
	o.SetSendBufInitSize(opt.SendBufInitSize)
	o.SetRecvBufMaxSize(opt.RecvBufInitSize)
	o.SetSendBufMaxSize(opt.SendBufMaxSize)
}
