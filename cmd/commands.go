package cmd

import "github.com/mitchellh/cli"

var Commands map[string]cli.CommandFactory

func init() {
	Commands = map[string]cli.CommandFactory{
		"target": func() (cli.Command, error) {
			return TargetCommand{}, nil
		},
	}
}
