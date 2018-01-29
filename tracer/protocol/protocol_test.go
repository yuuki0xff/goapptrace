package protocol

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProto_PackUnpack(t *testing.T) {
	a := assert.New(t)
	pkt := PingPacket{}
	proto := Proto{}
	pktData, err := proto.Pack(&pkt)
	if err != nil {
		t.Errorf("Proto.Pack() should not returns error: err=%s", err)
		return
	}

	persistPktSize := proto.PackSize(&pkt)
	if persistPktSize != len(pktData) {
		t.Errorf("PacketSize mismatch: PackSize()=%d actual=%d", persistPktSize, len(pktData))
	}

	pktSize := int(binary.BigEndian.Uint32(pktData[:4]))
	pktBody := pktData[4:]
	if pktSize != len(pktBody) {
		t.Errorf("Invalid PacketSize: persist=%d actual=%d", pktSize, len(pktBody))
	}

	buff := bytes.NewBuffer(nil)
	buff.Write(pktBody)

	hp := HeaderPacket{}
	pp := PingPacket{}

	a.NotPanics(func() {
		hp.Unmarshal(buff)
	}, "unmarshal of HeaderPacket")

	if hp.PacketType != PingPacketType {
		t.Errorf("PacketType is mismatch: expected=%d actual=%d", PingPacketType, hp.PacketType)
		return
	}

	a.NotPanics(func() {
		pp.Unmarshal(buff)
	}, "unmarshal of PingPacket")
}
