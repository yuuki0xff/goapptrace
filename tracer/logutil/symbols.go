package logutil

import (
	"log"
	"strconv"
	"strings"
)

func (f *FuncID) UnmarshalText(text []byte) error {
	id, err := strconv.ParseUint(string(text), 10, 64)
	if err != nil {
		return err
	}
	*f = FuncID(id)
	return nil
}

func (f *FuncStatusID) UnmarshalText(text []byte) error {
	id, err := strconv.ParseUint(string(text), 10, 64)
	if err != nil {
		return err
	}
	*f = FuncStatusID(id)
	return nil
}

func (s *Symbols) Init() {
	s.funcs = make([]*FuncSymbol, 0)
	s.funcStatus = make([]*FuncStatus, 0)
	if s.Writable {
		s.name2FuncID = make(map[string]FuncID)
		s.status2FSID = make(map[FuncStatus]FuncStatusID)
	}
}

func (s *Symbols) Load(funcs []*FuncSymbol, funcStatus []*FuncStatus) {
	s.Init()
	s.funcs = funcs
	s.funcStatus = funcStatus
}

func (s *Symbols) Save(fn func(funcs []*FuncSymbol, funcStatus []*FuncStatus) error) error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return fn(s.funcs, s.funcStatus)
}

func (s *Symbols) Func(id FuncID) (FuncSymbol, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if FuncID(len(s.funcs)) <= id {
		return FuncSymbol{}, false
	}
	f := s.funcs[id]
	if f == nil {
		return FuncSymbol{}, false
	}
	return *f, true
}

func (s *Symbols) FuncStatus(id FuncStatusID) (FuncStatus, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if FuncStatusID(len(s.funcStatus)) <= id {
		return FuncStatus{}, false
	}
	fs := s.funcStatus[id]
	if fs == nil {
		return FuncStatus{}, false
	}
	return *fs, true
}

func (s *Symbols) FuncsSize() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.funcs)
}

func (s *Symbols) FuncStatusSize() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.funcStatus)
}

func (s *Symbols) WalkFuncs(fn func(fs FuncSymbol) error) error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	for _, fs := range s.funcs {
		if fs != nil {
			if err := fn(*fs); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Symbols) WalkFuncStatus(fn func(fs FuncStatus) error) error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	for _, fs := range s.funcStatus {
		if fs != nil {
			if err := fn(*fs); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Symbols) FuncID(id FuncStatusID) FuncID {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.funcStatus[id].Func
}

func (s *Symbols) FuncName(id FuncStatusID) string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.funcs[s.funcStatus[id].Func].Name
}

func (s *Symbols) ModuleName(id FuncStatusID) string {
	funcName := s.FuncName(id)

	// strip function name from funcName
	moduleHierarchy := strings.Split(funcName, "/")
	last := len(moduleHierarchy) - 1
	moduleHierarchy[last] = strings.SplitN(moduleHierarchy[last], ".", 2)[0]
	moduleName := strings.Join(moduleHierarchy, "/")

	return moduleName
}

// 注意: 引数(symbols)のIDは引き継がれない。
func (s *Symbols) AddSymbols(symbols *Symbols) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, fsymbol := range symbols.funcs {
		s.addFuncNolock(fsymbol)
	}
	for _, fsatus := range symbols.funcStatus {
		s.addFuncStatusNolock(fsatus)
	}
}

func (s *Symbols) AddFunc(symbol *FuncSymbol) (id FuncID, added bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.addFuncNolock(symbol)
}

func (s *Symbols) addFuncNolock(symbol *FuncSymbol) (id FuncID, added bool) {
	if !s.Writable {
		log.Panic("Symbols is not writable")
	}

	id, ok := s.name2FuncID[symbol.Name]
	if ok {
		// if exists, nothing to do
		return id, false
	}

	if s.KeepID {
		// symbol.IDの値が、配列の長さを超えている場合、配列の長さを伸ばす。
		for symbol.ID >= FuncID(len(s.funcs)) {
			s.funcs = append(s.funcs, nil)
		}
	} else {
		symbol.ID = FuncID(len(s.funcs))
		// increase length of the funcs array
		s.funcs = append(s.funcs, nil)
	}
	s.funcs[symbol.ID] = symbol
	s.name2FuncID[symbol.Name] = symbol.ID
	return symbol.ID, true
}

func (s *Symbols) AddFuncStatus(status *FuncStatus) (id FuncStatusID, added bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.addFuncStatusNolock(status)
}

func (s *Symbols) addFuncStatusNolock(status *FuncStatus) (id FuncStatusID, added bool) {
	if !s.Writable {
		log.Panic("Symbols is not writable")
	}

	id, ok := s.status2FSID[*status]
	if ok {
		// if exists, nothing to do
		return id, false
	}

	if s.KeepID {
		// status.IDの値が配列の長さを超えている場合、配列の長さを伸ばす。
		for status.ID >= FuncStatusID(len(s.funcStatus)) {
			s.funcStatus = append(s.funcStatus, nil)
		}
	} else {
		status.ID = FuncStatusID(len(s.funcStatus))
		// increase length of the FuncStatus array
		s.funcStatus = append(s.funcStatus, status)
	}
	s.funcStatus[status.ID] = status
	s.status2FSID[*status] = status.ID
	return status.ID, true
}
