package target

import "fmt"

type LsCommand struct{}

func (LsCommand) Help() string {
	return "help"
}

func (LsCommand) Run(args []string) int {
	fmt.Println(args)
	return 0
}

func (LsCommand) Synopsis() string {
	return "show listing of tracing targets"
}
