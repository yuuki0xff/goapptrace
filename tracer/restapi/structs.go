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
	Logs []LogStatus `json:"logs"`
}

// TODO: aliasを削除する
type LogStatus types.LogInfo
type FuncCall = types.FuncLog
type FuncInfo = types.GoFunc
type GoLineInfo = types.GoLine

type SortOrder string
type SortKey string

type SearchFuncCallParams struct {
	Gid int64
	Fid int64
	//Mid          int64
	MinId        int64
	MaxId        int64
	MinTimestamp int64
	MaxTimestamp int64
	Limit        int64
	SortKey      SortKey
	SortOrder    SortOrder
}

// ToParamMap converts this to url parameters map.
func (s SearchFuncCallParams) ToParamMap() map[string]string {
	m := map[string]string{}
	if s.Gid != 0 {
		m["gid"] = strconv.Itoa(int(s.Gid))
	}
	if s.Fid != 0 {
		m["fid"] = strconv.Itoa(int(s.Fid))
	}
	//if s.Mid != 0 {
	//	m["mid" = strconv.Itoa(int(s.Mid))
	//}
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
	return m
}
