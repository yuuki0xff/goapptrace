package restapi

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"
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

type SortKey string

func (key *SortKey) Parse(s string) error {
	switch s {
	case string(NoSortKey):
		fallthrough
	case string(SortByID):
		fallthrough
	case string(SortByStartTime):
		fallthrough
	case string(SortByEndTime):
		*key = SortKey(s)
		return nil
	default:
		return fmt.Errorf("invalid sort key: %s", s)
	}
}

type SortOrder string

func (o *SortOrder) Parse(order string, defaultOrder SortOrder) error {
	switch order {
	case string(NoSortOrder):
		*o = defaultOrder
		return nil
	case string(AscendingSortOrder):
		fallthrough
	case string(DescendingSortOrder):
		*o = SortOrder(order)
		return nil
	default:
		return fmt.Errorf("invalid SortOrder: %s", order)
	}
}

type SearchFuncLogParams struct {
	Gid          types.GID
	MinId        types.FuncLogID
	MaxId        types.FuncLogID
	MinTimestamp types.Time
	MaxTimestamp types.Time
	Limit        int64
	SortKey      SortKey
	SortOrder    SortOrder
	Sql          string
}

// ToParamMap converts this to url parameters map.
func (s SearchFuncLogParams) ToParamMap() map[string]string {
	m := map[string]string{}
	if s.Gid != 0 {
		m["gid"] = s.Gid.String()
	}
	if s.MinId != 0 {
		m["min-id"] = s.MinId.String()
	}
	if s.MaxId != 0 {
		m["max-id"] = s.MaxId.String()
	}
	if s.MinTimestamp != 0 {
		m["min-timestamp"] = s.MinTimestamp.NumberString()
	}
	if s.MaxTimestamp != 0 {
		m["max-timestamp"] = s.MaxTimestamp.NumberString()
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

func (s *SearchFuncLogParams) FromString(
	gid, minId, maxId, minTs, maxTs, limit, sort, order, sql string,
) (invalidParamName string, err error) {
	defer func() {
		err = errors.Wrap(err, "invalid "+invalidParamName)
	}()
	tmp := SearchFuncLogParams{
		Gid:          -1,
		MinId:        -1,
		MaxId:        -1,
		MinTimestamp: -1,
		MaxTimestamp: -1,
		Limit:        -1,
		SortKey:      "",
		SortOrder:    AscendingSortOrder,
		Sql:          sql,
	}
	if gid != "" {
		err = tmp.Gid.FromString(gid)
		if err != nil {
			invalidParamName = "gid"
			return
		}
	}
	if minId != "" {
		err = tmp.MinId.FromString(minId)
		if err != nil {
			invalidParamName = "min-id"
			return
		}
	}
	if maxId != "" {
		err = tmp.MaxId.FromString(maxId)
		if err != nil {
			invalidParamName = "max-id"
			return
		}
	}
	if minTs != "" {
		err = tmp.MinTimestamp.FromNumberString(minTs)
		if err != nil {
			invalidParamName = "min-timestamp"
			return
		}
	}
	if maxTs != "" {
		err = tmp.MaxTimestamp.FromNumberString(maxTs)
		if err != nil {
			invalidParamName = "max-timestamp"
			return
		}
	}
	if limit != "" {
		tmp.Limit, err = strconv.ParseInt(limit, 10, 64)
		if err != nil {
			invalidParamName = "limit"
			return
		}
	}
	if sort != "" {
		err = tmp.SortKey.Parse(sort)
		if err != nil {
			invalidParamName = "sort"
			return
		}
	}
	if order != "" {
		err = tmp.SortOrder.Parse(order, AscendingSortOrder)
		if err != nil {
			invalidParamName = "order"
			return
		}
	}
	*s = tmp
	invalidParamName = ""
	err = nil
	return
}
