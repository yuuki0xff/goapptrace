package logger

import (
	"runtime"
)

var defaultIsTracing bool
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

// EnableAll starts all functions of goapptrace.
func EnableAll() {
	lock.Lock()
	defer lock.Unlock()
	for _, bp := range funcIsTracing {
		*bp = true
	}
	defaultIsTracing = true
}

// DisableAll stops all functions of goapptrace until calls EnableAll() or StartFunc().
func DisableAll() {
	lock.Lock()
	defer lock.Unlock()
	for _, bp := range funcIsTracing {
		*bp = false
	}
	defaultIsTracing = false
}

// EnableTrace enables tracing of specify function.
func EnableTrace(funcName string) {
	lock.Lock()
	defer lock.Unlock()
	if bp, ok := funcIsTracing[funcName]; ok {
		*bp = true
	} else {
		bp = new(bool)
		*bp = true
		funcIsTracing[funcName] = bp
	}
}

// DisableTrace disables tracing of specify function.
func DisableTrace(funcName string) {
	lock.Lock()
	defer lock.Unlock()
	if bp, ok := funcIsTracing[funcName]; ok {
		*bp = false
	} else {
		funcIsTracing[funcName] = new(bool)
	}
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
		*funcIsTracing[funcName] = defaultIsTracing
	}
	return funcIsTracing[funcName]
}
