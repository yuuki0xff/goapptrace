package config

const (
	DefaultLogServerAddr = "tcp://localhost:8600"
	DefaultApiServerAddr = "http://localhost:8700"
)

type ServerID int64
type Servers struct {
	LogServer map[ServerID]*LogServerConfig
	ApiServer map[ServerID]*ApiServerConfig
}

func NewServers() *Servers {
	return &Servers{
		LogServer: map[ServerID]*LogServerConfig{},
		ApiServer: map[ServerID]*ApiServerConfig{},
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
type ApiServerConfig struct {
	ServerID ServerID `json:"server-id"`
	Version  int      `json:"version"`
	// server address like "http://x.x.x.x:xxxx".
	Addr string `json:"address"`
}
