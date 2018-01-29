package protocol

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProto_PackTo_Unpack(t *testing.T) {
	a := assert.New(t)
	buff := &bytes.Buffer{}
	proto := Proto{}

	// PackTo
	sendPkt := PingPacket{}
	_, err := proto.PackTo(&sendPkt, buff)
	a.NoError(err, "Proto.PackTo()")

	pktData := buff.Bytes()
	pktSize := int(binary.BigEndian.Uint32(pktData[:4]))
	pktBody := pktData[4:]
	a.Equal(len(pktBody), pktSize)

	// UnpackTo
	recevPkt, n, err := proto.Unpack(pktData)
	a.NoError(err, "Proto.Unpack")
	a.Equal(len(pktData), n)
	a.Equal(&sendPkt, recevPkt)
}
