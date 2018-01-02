package config

type ServerID int64
type Servers struct {
	LogServer map[ServerID]*LogServerConfig
	ApiServer map[ServerID]*ApiServerConfg
}

// configuration for Log server.
type LogServerConfig struct {
	Addr string
}

// configuration for the REST API server.
type ApiServerConfg struct {
	Addr string
}
