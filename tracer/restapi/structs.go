package restapi

import (
	"strconv"

	"github.com/yuuki0xff/goapptrace/config"
	"github.com/yuuki0xff/goapptrace/tracer/types"
)

type Servers struct {
	Servers []ServerStatus `json:"servers"`
}
type ServerStatus config.LogServerConfig

type Logs struct {
	Logs []types.LogInfo `json:"logs"`
}

type SortOrder string
type SortKey string

type SearchFuncLogParams struct {
	Gid          int64
	MinId        int64
	MaxId        int64
	MinTimestamp int64
	MaxTimestamp int64
	Limit        int64
	SortKey      SortKey
	SortOrder    SortOrder
	Sql          string
}

// ToParamMap converts this to url parameters map.
func (s SearchFuncLogParams) ToParamMap() map[string]string {
	m := map[string]string{}
	if s.Gid != 0 {
		m["gid"] = strconv.Itoa(int(s.Gid))
	}
	if s.MinId != 0 {
		m["min-id"] = strconv.Itoa(int(s.MinId))
	}
	if s.MaxId != 0 {
		m["max-id"] = strconv.Itoa(int(s.MaxId))
	}
	if s.MinTimestamp != 0 {
		m["min-timestamp"] = strconv.Itoa(int(s.MinTimestamp))
	}
	if s.MaxTimestamp != 0 {
		m["max-timestamp"] = strconv.Itoa(int(s.MaxTimestamp))
	}
	if s.Limit != 0 {
		m["limit"] = strconv.FormatInt(s.Limit, 10)
	}
	if s.SortKey != NoSortKey {
		m["sort"] = string(s.SortKey)
	}
	if s.SortOrder != NoSortOrder {
		m["order"] = string(s.SortOrder)
	}
	if s.Sql != "" {
		m["sql"] = s.Sql
	}
	return m
}
