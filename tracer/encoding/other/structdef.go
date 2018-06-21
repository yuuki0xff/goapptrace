package goserbench

//go:generate msgp -o msgp_gen.go -io=false -tests=false
//easyjson:json
type A struct {
	ID        int64
	Tag       uint8
	Timestamp int64
	Frames    []uint64
	GID       int64
	TxID      uint64
}
