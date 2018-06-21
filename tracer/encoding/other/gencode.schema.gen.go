package goserbench

import (
	"io"
	"time"
	"unsafe"
)

var (
	_ = unsafe.Sizeof(0)
	_ = io.ReadFull
	_ = time.Now()
)

type GencodeA struct {
	ID        int64
	Tag       uint8
	Timestamp int64
	Frames    []uint64
	GID       int64
	TxID      uint64
}

func (d *GencodeA) Size() (s uint64) {

	{
		l := uint64(len(d.Frames))

		{

			t := l
			for t >= 0x80 {
				t >>= 7
				s++
			}
			s++

		}

		s += 8 * l

	}
	s += 33
	return
}
func (d *GencodeA) Marshal(buf []byte) ([]byte, error) {
	size := d.Size()
	{
		if uint64(cap(buf)) >= size {
			buf = buf[:size]
		} else {
			buf = make([]byte, size)
		}
	}
	i := uint64(0)

	{

		buf[0+0] = byte(d.ID >> 0)

		buf[1+0] = byte(d.ID >> 8)

		buf[2+0] = byte(d.ID >> 16)

		buf[3+0] = byte(d.ID >> 24)

		buf[4+0] = byte(d.ID >> 32)

		buf[5+0] = byte(d.ID >> 40)

		buf[6+0] = byte(d.ID >> 48)

		buf[7+0] = byte(d.ID >> 56)

	}
	{

		buf[0+8] = byte(d.Tag >> 0)

	}
	{

		buf[0+9] = byte(d.Timestamp >> 0)

		buf[1+9] = byte(d.Timestamp >> 8)

		buf[2+9] = byte(d.Timestamp >> 16)

		buf[3+9] = byte(d.Timestamp >> 24)

		buf[4+9] = byte(d.Timestamp >> 32)

		buf[5+9] = byte(d.Timestamp >> 40)

		buf[6+9] = byte(d.Timestamp >> 48)

		buf[7+9] = byte(d.Timestamp >> 56)

	}
	{
		l := uint64(len(d.Frames))

		{

			t := uint64(l)

			for t >= 0x80 {
				buf[i+17] = byte(t) | 0x80
				t >>= 7
				i++
			}
			buf[i+17] = byte(t)
			i++

		}
		for k0 := range d.Frames {

			{

				buf[i+0+17] = byte(d.Frames[k0] >> 0)

				buf[i+1+17] = byte(d.Frames[k0] >> 8)

				buf[i+2+17] = byte(d.Frames[k0] >> 16)

				buf[i+3+17] = byte(d.Frames[k0] >> 24)

				buf[i+4+17] = byte(d.Frames[k0] >> 32)

				buf[i+5+17] = byte(d.Frames[k0] >> 40)

				buf[i+6+17] = byte(d.Frames[k0] >> 48)

				buf[i+7+17] = byte(d.Frames[k0] >> 56)

			}

			i += 8

		}
	}
	{

		buf[i+0+17] = byte(d.GID >> 0)

		buf[i+1+17] = byte(d.GID >> 8)

		buf[i+2+17] = byte(d.GID >> 16)

		buf[i+3+17] = byte(d.GID >> 24)

		buf[i+4+17] = byte(d.GID >> 32)

		buf[i+5+17] = byte(d.GID >> 40)

		buf[i+6+17] = byte(d.GID >> 48)

		buf[i+7+17] = byte(d.GID >> 56)

	}
	{

		buf[i+0+25] = byte(d.TxID >> 0)

		buf[i+1+25] = byte(d.TxID >> 8)

		buf[i+2+25] = byte(d.TxID >> 16)

		buf[i+3+25] = byte(d.TxID >> 24)

		buf[i+4+25] = byte(d.TxID >> 32)

		buf[i+5+25] = byte(d.TxID >> 40)

		buf[i+6+25] = byte(d.TxID >> 48)

		buf[i+7+25] = byte(d.TxID >> 56)

	}
	return buf[:i+33], nil
}

func (d *GencodeA) Unmarshal(buf []byte) (uint64, error) {
	i := uint64(0)

	{

		d.ID = 0 | (int64(buf[i+0+0]) << 0) | (int64(buf[i+1+0]) << 8) | (int64(buf[i+2+0]) << 16) | (int64(buf[i+3+0]) << 24) | (int64(buf[i+4+0]) << 32) | (int64(buf[i+5+0]) << 40) | (int64(buf[i+6+0]) << 48) | (int64(buf[i+7+0]) << 56)

	}
	{

		d.Tag = 0 | (uint8(buf[i+0+8]) << 0)

	}
	{

		d.Timestamp = 0 | (int64(buf[i+0+9]) << 0) | (int64(buf[i+1+9]) << 8) | (int64(buf[i+2+9]) << 16) | (int64(buf[i+3+9]) << 24) | (int64(buf[i+4+9]) << 32) | (int64(buf[i+5+9]) << 40) | (int64(buf[i+6+9]) << 48) | (int64(buf[i+7+9]) << 56)

	}
	{
		l := uint64(0)

		{

			bs := uint8(7)
			t := uint64(buf[i+17] & 0x7F)
			for buf[i+17]&0x80 == 0x80 {
				i++
				t |= uint64(buf[i+17]&0x7F) << bs
				bs += 7
			}
			i++

			l = t

		}
		if uint64(cap(d.Frames)) >= l {
			d.Frames = d.Frames[:l]
		} else {
			d.Frames = make([]uint64, l)
		}
		for k0 := range d.Frames {

			{

				d.Frames[k0] = 0 | (uint64(buf[i+0+17]) << 0) | (uint64(buf[i+1+17]) << 8) | (uint64(buf[i+2+17]) << 16) | (uint64(buf[i+3+17]) << 24) | (uint64(buf[i+4+17]) << 32) | (uint64(buf[i+5+17]) << 40) | (uint64(buf[i+6+17]) << 48) | (uint64(buf[i+7+17]) << 56)

			}

			i += 8

		}
	}
	{

		d.GID = 0 | (int64(buf[i+0+17]) << 0) | (int64(buf[i+1+17]) << 8) | (int64(buf[i+2+17]) << 16) | (int64(buf[i+3+17]) << 24) | (int64(buf[i+4+17]) << 32) | (int64(buf[i+5+17]) << 40) | (int64(buf[i+6+17]) << 48) | (int64(buf[i+7+17]) << 56)

	}
	{

		d.TxID = 0 | (uint64(buf[i+0+25]) << 0) | (uint64(buf[i+1+25]) << 8) | (uint64(buf[i+2+25]) << 16) | (uint64(buf[i+3+25]) << 24) | (uint64(buf[i+4+25]) << 32) | (uint64(buf[i+5+25]) << 40) | (uint64(buf[i+6+25]) << 48) | (uint64(buf[i+7+25]) << 56)

	}
	return i + 33, nil
}
