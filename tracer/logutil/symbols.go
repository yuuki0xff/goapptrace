package logutil

import (
	"strings"
)

func (s *Symbols) Init() {
	s.Funcs = make([]*FuncSymbol, 0)
	s.FuncStatus = make([]*FuncStatus, 0)
}

func (s Symbols) FuncID(id FuncStatusID) FuncID {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.FuncStatus[id].Func
}

func (s Symbols) FuncName(id FuncStatusID) string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.Funcs[s.FuncID(id)].Name
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

func (sr *SymbolsEditor) Init(symbols *Symbols) {
	sr.symbols = symbols

	sr.symbols.lock.Lock()
	defer sr.symbols.lock.Unlock()

	sr.funcs = make(map[string]FuncID)
	sr.funcStatus = make(map[FuncStatus]FuncStatusID)
}

// 注意: 引数(symbols)のIDは引き継がれない。
func (sr *SymbolsEditor) AddSymbols(symbols *Symbols) {
	sr.symbols.lock.Lock()
	defer sr.symbols.lock.Unlock()

	for _, fsymbol := range symbols.Funcs {
		sr.addFuncNolock(fsymbol)
	}
	for _, fsatus := range symbols.FuncStatus {
		sr.addFuncStatusNolock(fsatus)
	}
}

func (sr *SymbolsEditor) AddFunc(symbol *FuncSymbol) (id FuncID, added bool) {
	sr.symbols.lock.Lock()
	defer sr.symbols.lock.Unlock()
	return sr.addFuncNolock(symbol)
}

func (sr *SymbolsEditor) addFuncNolock(symbol *FuncSymbol) (id FuncID, added bool) {
	id, ok := sr.funcs[symbol.Name]
	if ok {
		// if exists, nothing to do
		return id, false
	}

	if sr.KeepID {
		// symbol.IDの値が、配列の長さを超えている場合、配列の長さを伸ばす。
		for symbol.ID >= FuncID(len(sr.symbols.Funcs)) {
			sr.symbols.Funcs = append(sr.symbols.Funcs, nil)
		}
	} else {
		symbol.ID = FuncID(len(sr.symbols.Funcs))
		// increase length of Funcs array
		sr.symbols.Funcs = append(sr.symbols.Funcs, nil)
	}
	sr.symbols.Funcs[symbol.ID] = symbol
	sr.funcs[symbol.Name] = symbol.ID
	return symbol.ID, true
}

func (sr *SymbolsEditor) AddFuncStatus(status *FuncStatus) (id FuncStatusID, added bool) {
	sr.symbols.lock.Lock()
	defer sr.symbols.lock.Unlock()
	return sr.addFuncStatusNolock(status)
}

func (sr *SymbolsEditor) addFuncStatusNolock(status *FuncStatus) (id FuncStatusID, added bool) {
	id, ok := sr.funcStatus[*status]
	if ok {
		// if exists, nothing to do
		return id, false
	}

	if sr.KeepID {
		// status.IDの値が配列の長さを超えている場合、配列の長さを伸ばす。
		for status.ID >= FuncStatusID(len(sr.symbols.FuncStatus)) {
			sr.symbols.FuncStatus = append(sr.symbols.FuncStatus, nil)
		}
	} else {
		status.ID = FuncStatusID(len(sr.symbols.FuncStatus))
		// increase length of Funcs array
		sr.symbols.FuncStatus = append(sr.symbols.FuncStatus, status)
	}
	sr.symbols.FuncStatus[status.ID] = status
	sr.funcStatus[*status] = status.ID
	return status.ID, true
}
