package logutil

import (
	"log"
	"strings"
)

func (s *Symbols) Init(writable bool, keepID bool) {
	s.Funcs = make([]*FuncSymbol, 0)
	s.FuncStatus = make([]*FuncStatus, 0)
	s.isWritable = writable
	s.keepID = keepID
	if s.isWritable {
		s.name2FuncID = make(map[string]FuncID)
		s.status2FSID = make(map[FuncStatus]FuncStatusID)
	}
}

func (s Symbols) FuncID(id FuncStatusID) FuncID {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.FuncStatus[id].Func
}

func (s Symbols) FuncName(id FuncStatusID) string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.Funcs[s.FuncStatus[id].Func].Name
}

func (s Symbols) ModuleName(id FuncStatusID) string {
	funcName := s.FuncName(id)

	// strip function name from funcName
	moduleHierarchy := strings.Split(funcName, "/")
	last := len(moduleHierarchy) - 1
	moduleHierarchy[last] = strings.SplitN(moduleHierarchy[last], ".", 2)[0]
	moduleName := strings.Join(moduleHierarchy, "/")

	return moduleName
}

// 注意: 引数(symbols)のIDは引き継がれない。
func (sr *Symbols) AddSymbols(symbols *Symbols) {
	sr.lock.Lock()
	defer sr.lock.Unlock()

	for _, fsymbol := range symbols.Funcs {
		sr.addFuncNolock(fsymbol)
	}
	for _, fsatus := range symbols.FuncStatus {
		sr.addFuncStatusNolock(fsatus)
	}
}

func (sr *Symbols) AddFunc(symbol *FuncSymbol) (id FuncID, added bool) {
	sr.lock.Lock()
	defer sr.lock.Unlock()
	return sr.addFuncNolock(symbol)
}

func (sr *Symbols) addFuncNolock(symbol *FuncSymbol) (id FuncID, added bool) {
	if !sr.isWritable {
		log.Panic("Symbols is not writable")
	}

	id, ok := sr.name2FuncID[symbol.Name]
	if ok {
		// if exists, nothing to do
		return id, false
	}

	if sr.keepID {
		// symbol.IDの値が、配列の長さを超えている場合、配列の長さを伸ばす。
		for symbol.ID >= FuncID(len(sr.Funcs)) {
			sr.Funcs = append(sr.Funcs, nil)
		}
	} else {
		symbol.ID = FuncID(len(sr.Funcs))
		// increase length of Funcs array
		sr.Funcs = append(sr.Funcs, nil)
	}
	sr.Funcs[symbol.ID] = symbol
	sr.name2FuncID[symbol.Name] = symbol.ID
	return symbol.ID, true
}

func (sr *Symbols) AddFuncStatus(status *FuncStatus) (id FuncStatusID, added bool) {
	sr.lock.Lock()
	defer sr.lock.Unlock()
	return sr.addFuncStatusNolock(status)
}

func (sr *Symbols) addFuncStatusNolock(status *FuncStatus) (id FuncStatusID, added bool) {
	if !sr.isWritable {
		log.Panic("Symbols is not writable")
	}

	id, ok := sr.status2FSID[*status]
	if ok {
		// if exists, nothing to do
		return id, false
	}

	if sr.keepID {
		// status.IDの値が配列の長さを超えている場合、配列の長さを伸ばす。
		for status.ID >= FuncStatusID(len(sr.FuncStatus)) {
			sr.FuncStatus = append(sr.FuncStatus, nil)
		}
	} else {
		status.ID = FuncStatusID(len(sr.FuncStatus))
		// increase length of Funcs array
		sr.FuncStatus = append(sr.FuncStatus, status)
	}
	sr.FuncStatus[status.ID] = status
	sr.status2FSID[*status] = status.ID
	return status.ID, true
}
