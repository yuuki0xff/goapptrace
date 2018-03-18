package storage

import (
	"github.com/yuuki0xff/goapptrace/tracer/encoding"
	"github.com/yuuki0xff/goapptrace/tracer/types"
)

type RawFuncLogStore struct {
	Store
}

func (s *RawFuncLogStore) Get(id types.RawFuncLogID, raw *types.RawFuncLog) error {
	s.Lock()
	defer s.Unlock()
	return s.GetNolock(id, raw)
}
func (s *RawFuncLogStore) Set(raw *types.RawFuncLog) error {
	s.Lock()
	defer s.Unlock()
	return s.SetNolock(raw)
}

func (s *RawFuncLogStore) GetNolock(id types.RawFuncLogID, raw *types.RawFuncLog) error {
	return s.ReadNolock(int64(id), func(buf []byte) {
		encoding.UnmarshalRawFuncLog(buf, raw)
	})
}
func (s *RawFuncLogStore) SetNolock(raw *types.RawFuncLog) error {
	return s.WriteNolock(int64(raw.ID), func(buf []byte) int64 {
		return encoding.MarshalRawFuncLog(buf, raw)
	})
}

type FuncLogStore struct {
	Store
}

func (s *FuncLogStore) Get(id types.FuncLogID, fl *types.FuncLog) error {
	s.Lock()
	defer s.Unlock()
	return s.GetNolock(id, fl)
}
func (s *FuncLogStore) Set(fl *types.FuncLog) error {
	s.Lock()
	defer s.Unlock()
	return s.SetNolock(fl)
}

func (s *FuncLogStore) GetNolock(id types.FuncLogID, fl *types.FuncLog) error {
	return s.ReadNolock(int64(id), func(buf []byte) {
		encoding.UnmarshalFuncLog(buf, fl)
	})
}
func (s *FuncLogStore) SetNolock(fl *types.FuncLog) error {
	return s.WriteNolock(int64(fl.ID), func(buf []byte) int64 {
		return encoding.MarshalFuncLog(buf, fl)
	})
}

type GoroutineStore struct {
	Store
}

func (s *GoroutineStore) Get(gid types.GID, g *types.Goroutine) error {
	s.Lock()
	defer s.Unlock()
	return s.GetNolock(gid, g)
}
func (s *GoroutineStore) Set(g *types.Goroutine) error {
	s.Lock()
	defer s.Unlock()
	return s.SetNolock(g)
}

func (s *GoroutineStore) GetNolock(gid types.GID, g *types.Goroutine) error {
	return s.ReadNolock(int64(gid), func(buf []byte) {
		encoding.UnmarshalGoroutine(buf, g)
	})
}
func (s *GoroutineStore) SetNolock(g *types.Goroutine) error {
	return s.WriteNolock(int64(g.GID), func(buf []byte) int64 {
		return encoding.MarshalGoroutine(buf, g)
	})
}
