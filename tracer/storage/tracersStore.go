package storage

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/yuuki0xff/goapptrace/tracer/types"
)

type TracersStoreUpdateFn func(tracer *types.Tracer) error

type TracersStore struct {
	File     File
	initOnce sync.Once
	m        sync.RWMutex
	tracers  []*types.Tracer
}

func (s *TracersStore) init() (err error) {
	s.initOnce.Do(func() {
		err = s.load()
	})
	if err != nil {
		s.tracers = nil
	}
	return
}
func (s *TracersStore) load() error {
	var js []byte
	js, err := s.File.ReadAll()
	if err != nil {
		return err
	}
	return json.Unmarshal(js, &s.tracers)
}
func (s *TracersStore) save() error {
	js, err := json.Marshal(s.tracers)
	if err != nil {
		return err
	}
	return s.File.WriteAll(js)
}
func (s *TracersStore) lookupById(id int) int {
	for i, t := range s.tracers {
		if t.ID == id {
			return i
		}
	}
	return -1
}
func (s *TracersStore) Get(id int) (*types.Tracer, error) {
	s.m.RLock()
	defer s.m.RUnlock()
	if err := s.init(); err != nil {
		return nil, err
	}

	idx := s.lookupById(id)
	if idx < 0 {
		return nil, nil
	}
	return s.tracers[idx], nil
}
func (s *TracersStore) Update(id int, fn TracersStoreUpdateFn) error {
	s.m.Lock()
	defer s.m.Unlock()

	if err := s.init(); err != nil {
		return err
	}

	idx := s.lookupById(id)
	if idx < 0 {
		return fmt.Errorf("not found Tracer(id=%d)", id)
	}

	t := &types.Tracer{}
	s.tracers[idx].Copy(t)
	if err := fn(t); err != nil {
		return err
	}
	s.tracers[idx] = t

	return s.save()
}
