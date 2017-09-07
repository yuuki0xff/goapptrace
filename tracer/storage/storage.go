package storage

import (
	"crypto/rand"
	"io/ioutil"
	"sync"
)

type Storage struct {
	Root DirLayout

	lock  sync.RWMutex
	files map[LogID]*Log
}

// Return all log instances
func (s *Storage) Logs() ([]*Log, error) {
	files, err := ioutil.ReadDir(s.Root.MetaDir())
	if err != nil {
		return nil, err
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	logs := make([]*Log, 0, len(files))
	for _, finfo := range files {
		id, ok := s.Root.MetaID(finfo.Name())
		if !ok {
			continue
		}

		log, ok := s.files[id]
		if !ok {
			log = s.log(id)
		}
		logs = append(logs, log)
	}
	return logs, nil
}

// 新しいLogインスタンスを作成する。
// この関数を呼び出す前に、排他ロックをかける必要がある。
func (s *Storage) log(id LogID) *Log {
	log := &Log{
		ID:   id,
		Root: s.Root,
	}
	s.files[id] = log
	return log
}

func (s *Storage) New() *Log {
	s.lock.Lock()
	defer s.lock.Unlock()

	// generate random id
	id := LogID{}
	if _, err := rand.Read(id[:]); err != nil {
		panic(err)
	}

	return s.log(id)
}
