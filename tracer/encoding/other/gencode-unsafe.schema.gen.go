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

type GencodeUnsafeA struct {
	ID        int64
	Tag       uint8
	Timestamp int64
	Frames    []uint64
	GID       int64
	TxID      uint64
}

func (d *GencodeUnsafeA) Size() (s uint64) {

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
func (d *GencodeUnsafeA) Marshal(buf []byte) ([]byte, error) {
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

		*(*int64)(unsafe.Pointer(&buf[0])) = d.ID

	}
	{

		*(*uint8)(unsafe.Pointer(&buf[8])) = d.Tag

	}
	{

		*(*int64)(unsafe.Pointer(&buf[9])) = d.Timestamp

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

				*(*uint64)(unsafe.Pointer(&buf[i+17])) = d.Frames[k0]

			}

			i += 8

		}
	}
	{

		*(*int64)(unsafe.Pointer(&buf[i+17])) = d.GID

	}
	{

		*(*uint64)(unsafe.Pointer(&buf[i+25])) = d.TxID

	}
	return buf[:i+33], nil
}

func (d *GencodeUnsafeA) Unmarshal(buf []byte) (uint64, error) {
	i := uint64(0)

	{

		d.ID = *(*int64)(unsafe.Pointer(&buf[i+0]))

	}
	{

		d.Tag = *(*uint8)(unsafe.Pointer(&buf[i+8]))

	}
	{

		d.Timestamp = *(*int64)(unsafe.Pointer(&buf[i+9]))

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

				d.Frames[k0] = *(*uint64)(unsafe.Pointer(&buf[i+17]))

			}

			i += 8

		}
	}
	{

		d.GID = *(*int64)(unsafe.Pointer(&buf[i+17]))

	}
	{

		d.TxID = *(*uint64)(unsafe.Pointer(&buf[i+25]))

	}
	return i + 33, nil
}
