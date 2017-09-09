package log

import "strings"

func (s Symbols) FuncID(id FuncStatusID) FuncID {
	return s.FuncStatus[id].Func
}

func (s Symbols) FuncName(id FuncStatusID) string {
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

func (sr *SymbolResolver) Init(symbols *Symbols) {
	if sr.funcs == nil {
		sr.symbols = symbols
		sr.funcs = make(map[string]FuncID)
		sr.funcStatus = make(map[FuncStatus]FuncStatusID)
	}
}

func (sr *SymbolResolver) AddFunc(symbol FuncSymbol) FuncID {
	id, ok := sr.funcs[symbol.Name]
	if ok {
		// if exists, nothing to do
		return id
	}

	symbol.ID = FuncID(len(sr.symbols.Funcs))
	sr.symbols.Funcs = append(sr.symbols.Funcs, &symbol)
	sr.funcs[symbol.Name] = symbol.ID
	return symbol.ID
}

func (sr *SymbolResolver) AddFuncStatus(status FuncStatus) FuncStatusID {
	status.ID = 0
	id, ok := sr.funcStatus[status]
	if ok {
		// if exists, nothing to do
		return id
	}

	// NOTE: sr.funcStatusのkeyとなるFuncStatusオブジェクトは、必ずFuncStatus.ID=0
	//       そのため、FuncStatus.IDを更新する前にsr.funcStatusへ追加する必要がある。
	id = FuncStatusID(len(sr.symbols.FuncStatus))
	sr.funcStatus[status] = id

	status.ID = id
	sr.symbols.FuncStatus = append(sr.symbols.FuncStatus, &status)
	return id
}
