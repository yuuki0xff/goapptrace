package storage

type Version uint64

// このプログラムが対応しているファイルフォーマットのバージョン
const (
	MajorVersion Version = 0
	MinorVersion Version = 0
)

// 現在参照しているファイルフォーマットのバージョン
type Info struct {
	MajorVersion Version
	MinorVersion Version
}

// このプログラムが対応しているバージョンであればtrueを返す。
func (i Info) IsCompatible() bool {
	return i.MajorVersion == MajorVersion
}
