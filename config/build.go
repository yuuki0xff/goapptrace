package config

import (
	"os"
	"os/exec"

	"github.com/yuuki0xff/goapptrace/info"
)

type BuildProcess struct {
	Args []string
}

func (bp *BuildProcess) Run() (*exec.Cmd, error) {
	args := bp.Args
	if args == nil || len(args) == 0 {
		args = []string{"go", "build", "-o", info.DEFAULT_EXE_NAME}
	}

	return runCmd(args)
}

func runCmd(args []string) (*exec.Cmd, error) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return cmd, nil
}
