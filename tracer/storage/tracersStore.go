package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"

	"github.com/yuuki0xff/goapptrace/tracer/types"
)

type TracersStoreUpdateFn func(tracer *types.Tracer) error

type TracersStore struct {
	File     File
	initOnce sync.Once
	m        sync.RWMutex
	tracers  []*types.Tracer
	// TracersStore に保存された情報が更新された場合にcallbackされる関数
	updateCallbacks map[int]func()
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
	err = json.Unmarshal(js, &s.tracers)
	if err != nil {
		return err
	}
	s.notify()
	return nil
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

// notify は、Watch()で登録されたコールバック関数を非同期的に呼び出す。
// TracersStore のデータが更新されたときに必ず呼び出すこと。
func (s *TracersStore) notify() {
	for _, fn := range s.updateCallbacks {
		go fn()
	}
}
func (s *TracersStore) Add() (*types.Tracer, error) {
	s.m.Lock()
	defer s.m.Unlock()

	t := &types.Tracer{
		ID: len(s.tracers),
	}
	s.tracers = append(s.tracers, t)
	if err := s.save(); err != nil {
		return nil, err
	}
	s.notify()
	return t, nil
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
func (s *TracersStore) GetAll() ([]*types.Tracer, error) {
	s.m.RLock()
	defer s.m.RUnlock()
	if err := s.init(); err != nil {
		return nil, err
	}
	return s.tracers, nil
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

	s.notify()
	return s.save()
}

// データが更新されたときにコールバックされる関数を登録する。
// callback()は、ctxが終了するまでこの関数は非同期的に呼び出される可能性がある。
func (s *TracersStore) Watch(ctx context.Context, callback func()) {
	var key int

	s.m.Lock()
	for {
		// updateCallbacksのキーの重複を避けるために乱数を使用している。
		// 乱数の品質は問わない(脆弱でも構わない)ため、gasのwarningを無視する。
		key = rand.Int() // nolint: gas
		if _, ok := s.updateCallbacks[key]; ok {
			continue
		}
		s.updateCallbacks[key] = callback
		break
	}
	s.m.Unlock()

	go func() {
		<-ctx.Done()
		s.m.Lock()
		delete(s.updateCallbacks, key)
		s.m.Unlock()
	}()
}
