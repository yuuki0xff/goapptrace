package storage

type Version uint64

const (
	MajorVersion Version = 0
	MinorVersion Version = 0
)

type Info struct {
	MajorVersion Version
	MinorVersion Version
}

func (i Info) IsCompatible() bool {
	return i.MajorVersion == MajorVersion
}
