package cmd

import (
	"github.com/yuuki0xff/goapptrace/cmd/target"
	"log"
)

type TargetCommand struct{}

func (TargetCommand) Help() string {
	return Help(
		[]string{"target"},
		target.Commands,
		map[string]string{},
	)
}

func (tc TargetCommand) Run(args []string) int {
	ret, err := GetSubCLI(tc, args, target.Commands).Run()
	if err != nil {
		log.Println(err)
	}
	return ret
}
func (TargetCommand) Synopsis() string {
	return "control tracing targets"
}
