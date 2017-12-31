package storage

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io/ioutil"
	"sync"
)

// ログの管理を行う。
type Storage struct {
	Root     DirLayout
	ReadOnly bool

	lock  sync.RWMutex
	files map[LogID]*Log
}

// 初期化を行う。使用前に必ず実行すること。
func (s *Storage) Init() error {
	s.files = make(map[LogID]*Log)

	if err := s.Root.Init(); err != nil {
		return err
	}
	return s.Load()
}

// ログファイルの一覧(Storage.files)の初期化を行う。
func (s *Storage) Load() error {
	files, err := ioutil.ReadDir(s.Root.MetaDir())
	if err != nil {
		return errors.New(fmt.Sprintln("Storage.Load(): failed get file list:", err.Error()))
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	for _, finfo := range files {
		id, ok := s.Root.Fname2LogID(finfo.Name())
		if !ok {
			continue
		}

		_, ok = s.files[id]
		if !ok {
			s.files[id], err = s.log(id, false)
			if err != nil {
				return errors.New(fmt.Sprintln("Storage.Load():", err))
			}
		}
	}
	return nil
}

// 全てのログを閉じる。
// これ以降、管理下のログへのアクセスは出来ない。
func (s *Storage) Close() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	for _, log := range s.files {
		if err := log.Close(); err != nil {
			return err
		}
	}
	return nil
}

// Return all log instances
func (s *Storage) Logs() ([]*Log, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	logs := make([]*Log, 0, len(s.files))
	for _, log := range s.files {
		logs = append(logs, log)
	}
	return logs, nil
}

// Return an exists log instance
func (s *Storage) Log(id LogID) (log *Log, ok bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	log, ok = s.files[id]
	return
}

// 新しいLogインスタンスを作成する。
// この関数を呼び出す前に、排他ロックをかける必要がある。
func (s *Storage) log(id LogID, new bool) (*Log, error) {
	log := &Log{
		ID:       id,
		Root:     s.Root,
		ReadOnly: s.ReadOnly,
	}
	if err := log.Open(); err != nil {
		return nil, fmt.Errorf("failed to open of Log(%s): %s", id.Hex(), err.Error())
	}
	s.files[id] = log
	return log, nil
}

// 新しいログインスタンスを作成して返す。
func (s *Storage) New() (*Log, error) {
	if s.ReadOnly {
		return nil, errors.New("cannot create a log on read-only storage")
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	id := LogID{}
	if _, err := rand.Read(id[:]); err != nil {
		return nil, err
	}

	return s.log(id, true)
}
