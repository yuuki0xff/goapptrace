package config

import (
	"os/exec"
	"path/filepath"

	"github.com/yuuki0xff/goapptrace/info"
)

type ExecProcess struct {
	Args []string
}

// Start execute a command but does not wait for exit
func (ep *ExecProcess) Start(env []string) (*exec.Cmd, error) {
	args := ep.Args
	if len(args) == 0 {
		args = []string{
			// NOTE: filepath.Join() will strip of "./"
			"." + string(filepath.Separator) + info.DefaultExeName,
		}
	}
	return startCmd(args, env)
}
