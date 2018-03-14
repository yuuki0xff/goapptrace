package protocol

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuuki0xff/xtcp"
)

var (
	pingPkt      = &PingPacket{}
	pingPktBytes = []byte{
		// length
		0, 0, 0, 1,
		// header packet data
		// NOTE: PingPacketType is 4.
		4,
		// ping packet data is empty.
	}

	fakePkt = &fakePacket{
		s: "fakePkt",
	}
	fakePktBytes = []byte{
		// length
		0, 0, 0, 8,
		// header packet data
		// NOTE: fakePacketType is 0.
		0,
		// packet data
		0x66, 0x61, 0x6b, 0x65, 0x50, 0x6b, 0x74,
	}
)

func TestProto_PackSize(t *testing.T) {
	a := assert.New(t)
	p := Proto{}

	a.Panics(func() {
		p.PackSize(&PingPacket{})
	})
}

func TestProto_PackTo(t *testing.T) {
	test := func(name string, pkt xtcp.Packet, b []byte) {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			a := assert.New(t)
			p := Proto{}

			n, err := p.PackTo(pkt, &buf)
			a.Equal(len(b), n)
			a.NoError(err)
			a.Equal(b, buf.Bytes())
		})
	}

	test("pingPacket", pingPkt, pingPktBytes)
	test("fakePacket", fakePkt, fakePktBytes)
}

func TestProto_Pack(t *testing.T) {
	a := assert.New(t)
	p := Proto{}

	a.Panics(func() {
		p.Pack(&PingPacket{})
	})
}

func TestProto_Unpack(t *testing.T) {
	test := func(name string, pkt xtcp.Packet, b []byte) {
		t.Run(name, func(t *testing.T) {
			a := assert.New(t)
			p := Proto{}

			pktUnpacked, n, err := p.Unpack(b)
			a.NoError(err)
			a.Equal(len(b), n)
			a.Equal(pkt, pktUnpacked)
		})
	}

	test("pingPacket", pingPkt, pingPktBytes)
	test("fakePacket", fakePkt, fakePktBytes)
}
