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

func (s *Storage) Init() error {
	s.files = make(map[LogID]*Log)
	return s.Root.Init()
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
			log = s.log(id, false)
		}
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
func (s *Storage) log(id LogID, new bool) *Log {
	log := &Log{
		ID:   id,
		Root: s.Root,
	}
	if new {
		if err := log.New(); err != nil {
			// TODO: return err
			panic(err)
		}
	} else {
		if err := log.Init(); err != nil {
			// TODO: return err
			panic(err)
		}
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

	return s.log(id, true)
}
