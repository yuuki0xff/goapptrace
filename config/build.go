package config

import (
	"os"
	"os/exec"

	"github.com/yuuki0xff/goapptrace/info"
)

type BuildProcess struct {
	Args []string
}

// Run run a command for build the app
// This method wait for exit build process.
func (bp *BuildProcess) Run() (*exec.Cmd, error) {
	args := bp.Args
	if len(args) == 0 {
		args = []string{"go", "build", "-o", info.DEFAULT_EXE_NAME}
	}

	return runCmd(args, nil)
}

// runCmd execute a command and wait for exit
// If env is nil, the new process uses the current process's
// environment.
func runCmd(args []string, env []string) (*exec.Cmd, error) {
	cmd, err := startCmd(args, env)
	if err != nil {
		return nil, err
	}
	return cmd, cmd.Wait()
}

// startCmd execute a command but does not wait for exit.
// If env is nil, the new process uses the current process's
// environment.
func startCmd(args []string, env []string) (*exec.Cmd, error) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env
	return cmd, cmd.Start()
}
