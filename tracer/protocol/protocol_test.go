package protocol

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestProto_PackUnpack(t *testing.T) {
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

	if err = hp.Unmarshal(buff); err != nil {
		t.Errorf("Should not occurs error when deserializing of HeaderPacket: err=%s", err)
		return
	}
	if hp.PacketType != PingPacketType {
		t.Errorf("PacketType is mismatch: expected=%d actual=%d", PingPacketType, hp.PacketType)
		return
	}

	if err = pp.Unmarshal(buff); err != nil {
		t.Errorf("Should not occurse error when deserializing of PingPacket: err=%s", err)
		return
	}
}
