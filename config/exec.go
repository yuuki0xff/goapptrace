package config

import (
	"path/filepath"

	"github.com/yuuki0xff/goapptrace/info"
)

type ExecProcess struct {
	Args []string
}

func (ep *ExecProcess) Run() error {
	args := ep.Args
	if args == nil || len(args) == 0 {
		args = []string{
			// NOTE: filepath.Join() will strip of "./"
			"." + string(filepath.Separator) + info.DEFAULT_EXE_NAME,
		}
	}
	return execCmd(args)
}
