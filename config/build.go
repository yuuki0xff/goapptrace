package config

import "os/exec"

type BuildProcess struct {
	Args []string
}

func (bp *BuildProcess) Run() error {
	cmd := exec.Command(bp.Args[0], bp.Args[1:])
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
