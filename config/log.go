package config

type Logs struct {
	logs []*Log
}

type Log struct {
	Start     int64 // Unix time
	End       int64 // Unix time
	File      string
	IsWriting bool
}
