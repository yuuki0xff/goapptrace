package config

const (
	DefaultLogServerAddr = "localhost:8600"
	DefaultApiServerAddr = "localhost:8700"
)

type ServerID int64
type Servers struct {
	LogServer map[ServerID]*LogServerConfig
	ApiServer map[ServerID]*ApiServerConfg
}

func NewServers() *Servers {
	return &Servers{
		LogServer: map[ServerID]*LogServerConfig{},
		ApiServer: map[ServerID]*ApiServerConfg{},
	}
}

// configuration for Log server.
type LogServerConfig struct {
	ServerID ServerID `json:"server-id"`
	Version  int      `json:"version"`
	// server address like "tcp://x.x.x.x:xxxx".
	Addr string `json:"address"`
}

// configuration for the REST API server.
type ApiServerConfg struct {
	ServerID ServerID `json:"server-id"`
	Version  int      `json:"version"`
	// server address like "http://x.x.x.x:xxxx".
	Addr string `json:"address"`
}
