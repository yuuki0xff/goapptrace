package cmd

import (
	"fmt"
	"github.com/yuuki0xff/goapptrace/config"
)

func target_ls(args CommandArgs) (int, error) {
	fmt.Println("ls")

	if err := args.Config.Targets.Walk(func(t *config.Target) error {
		for i := range t.Files {
			fmt.Printf("%s  %s\n", t.Name, t.Files[i])
		}
		return nil
	}); err != nil {
		return 1, err
	}
	return 0, nil
}

func target_add(args CommandArgs) (int, error) {
	if err := args.Config.Targets.Add(&config.Target{
		Name:  config.TargetName(args.Arguments[0]),
		Files: args.Arguments[1:],
	}); err != nil {
		return 1, err
	}
	return 0, nil
}

func target_remove(args CommandArgs) (int, error) {
	for _, name := range args.Arguments {
		if err := args.Config.Targets.Delete(config.TargetName(name)); err != nil {
			return 1, err
		}
	}
	return 0, nil
}

func target_set_build(args CommandArgs) (int, error) {
	name, cmds := args.Arguments[0], args.Arguments[1:]
	t, err := args.Config.Targets.Get(config.TargetName(name))
	if err != nil {
		return 1, err
	}
	t.Build.Args = cmds
	return 0, nil
}
