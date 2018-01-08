package restapi

import (
	"encoding/json"

	"github.com/yuuki0xff/goapptrace/config"
)

type ServersResponse struct {
	Servers []*config.LogServerConfig `json:"servers"`
}

type LogsResponse struct {
	Logs []json.RawMessage `json:"logs"`
}
