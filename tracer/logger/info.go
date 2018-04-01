package logger

import (
	"runtime"
)

var pkgs []*PackageData

type PackageData struct {
	Name  string
	Funcs []FuncConfig
}
type FuncConfig struct {
	Name      string
	IsTracing *bool
}

func AddPackage(data *PackageData) {
	pkgs = append(pkgs, data)
}

func setIsTracingFlag(match func(config FuncConfig) bool, flag bool) {
	for _, pkg := range pkgs {
		for _, f := range pkg.Funcs {
			if match(f) {
				*f.IsTracing = flag
				break
			}
		}
	}
}

// EnableAll starts all functions of goapptrace.
func EnableAll() {
	setIsTracingFlag(func(FuncConfig) bool { return true }, true)
}

// DisableAll stops all functions of goapptrace until calls EnableAll() or StartFunc().
func DisableAll() {
	setIsTracingFlag(func(FuncConfig) bool { return true }, false)
}

// EnableTrace enables tracing of specify function.
func EnableTrace(funcName string) {
	setIsTracingFlag(func(config FuncConfig) bool { return config.Name == funcName }, true)
}

// DisableTrace disables tracing of specify function.
func DisableTrace(funcName string) {
	setIsTracingFlag(func(config FuncConfig) bool { return config.Name == funcName }, false)
}

func TracingFlag() *bool {
	lock.Lock()
	defer lock.Unlock()

	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		panic("bug")
	}
	f, ok := symbols.GoFunc(pc)
	var funcName string
	if ok {
		funcName = f.Name
	} else {
		// symbolsが初期化されていない状態では、runtimeから関数名を取得する。
		funcObj := runtime.FuncForPC(pc)
		if funcObj == nil {
			panic("bug")
		}
		funcName = funcObj.Name()
	}
	if funcIsTracing[funcName] == nil {
		funcIsTracing[funcName] = new(bool)
	}
	return funcIsTracing[funcName]
}
