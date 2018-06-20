package goserbench

//go:generate -command genxdr go run ../../calmh/xdr/cmd/genxdr/main.go
//go:generate genxdr -o structdefxdr_generated.go structdefxdr.go
type XDRA struct {
	ID        int64
	Tag       uint8
	Timestamp int64
	Frames    []uint64
	GID       int64
	TxID      uint64
}
