package cmd

import (
	"github.com/kesselborn/go-getopt"
	"github.com/yuuki0xff/goapptrace/config"
)

type CommandArgs struct {
	Options   map[string]getopt.OptionValue
	Arguments []string
	Config    *config.Config
}

type CommandFn func(args CommandArgs) (int, error)

var cmd = map[string]map[string]CommandFn{
	"target": {
		"ls":        target_ls,
		"add":       target_add,
		"remove":    target_remove,
		"set-build": target_set_build,
	},
	"trace": {
		"on":     trace_on,
		"off":    trace_off,
		"status": trace_status,
		"start":  trace_start,
		"stop":   trace_stop,
	},
}

func Get(scope, subCmd string) (CommandFn, bool) {
	a, exists := cmd[scope]
	if !exists {
		return nil, false
	}
	b, exists := a[subCmd]
	return b, exists
}
