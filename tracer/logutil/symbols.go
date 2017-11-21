package logutil

import "strings"

func (s *Symbols) Init() {
	s.Funcs = make([]*FuncSymbol, 0)
	s.FuncStatus = make([]*FuncStatus, 0)
}

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

func (sr *SymbolsEditor) Init(symbols *Symbols) {
	sr.symbols = symbols
	sr.funcs = make(map[string]FuncID)
	sr.funcStatus = make(map[FuncStatus]FuncStatusID)
}

// 注意: 引数(symbols)のIDは引き継がれない。
func (sr *SymbolsEditor) AddSymbols(symbols *Symbols) {
	for _, fsymbol := range symbols.Funcs {
		sr.AddFunc(fsymbol)
	}
	for _, fsatus := range symbols.FuncStatus {
		sr.AddFuncStatus(fsatus)
	}
}

func (sr *SymbolsEditor) AddFunc(symbol *FuncSymbol) (id FuncID, added bool) {
	id, ok := sr.funcs[symbol.Name]
	if ok {
		// if exists, nothing to do
		return id, false
	}

	symbol.ID = FuncID(len(sr.symbols.Funcs))
	sr.symbols.Funcs = append(sr.symbols.Funcs, symbol)
	sr.funcs[symbol.Name] = symbol.ID
	return symbol.ID, true
}

func (sr *SymbolsEditor) AddFuncStatus(status *FuncStatus) (id FuncStatusID, added bool) {
	status.ID = 0
	id, ok := sr.funcStatus[*status]
	if ok {
		// if exists, nothing to do
		return id, false
	}

	// NOTE: sr.funcStatusのkeyとなるFuncStatusオブジェクトは、必ずFuncStatus.ID=0
	//       そのため、FuncStatus.IDを更新する前にsr.funcStatusへ追加する必要がある。
	id = FuncStatusID(len(sr.symbols.FuncStatus))
	sr.funcStatus[*status] = id

	status.ID = id
	sr.symbols.FuncStatus = append(sr.symbols.FuncStatus, status)
	return id, true
}
