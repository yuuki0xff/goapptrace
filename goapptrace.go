package main

import (
	"fmt"
	"github.com/kesselborn/go-getopt"
	"github.com/yuuki0xff/goapptrace/config"
	"github.com/yuuki0xff/goapptrace/info"
	"os"
)

func main() {
	os.Exit(realMain())
}

func realMain() int {
	ssco := getopt.SubSubCommandOptions{
		Global: getopt.Options{
			Description: "function calls tracer for golang",
			Definitions: getopt.Definitions{
				{"config|c", "set a config dir path instead of '.goapptrace/'", getopt.Optional, ""},
				{"scope", "", getopt.IsSubCommand, ""},
			},
		},
		Scopes: getopt.Scopes{
			"target": {
				getopt.Options{
					"control tracing targets",
					getopt.Definitions{
						{"command", "command to execute", getopt.IsSubCommand, ""},
					},
				},
				getopt.SubCommands{
					"ls": {
						"show tracing targets",
						getopt.Definitions{
							{"name", "target name", getopt.Optional | getopt.IsArg, ""},
						},
					},
					"add": {
						"add tracing targets",
						getopt.Definitions{
							{"name", "target name", getopt.IsArg | getopt.Required, ""},
							{"targets", "dirs or files", getopt.IsArg | getopt.Required, ""},
						},
					},
					"remove": {
						"remove from tracing targets",
						getopt.Definitions{
							{"name", "target name", getopt.IsArg | getopt.Required, ""},
							{"targets", "dirs or files", getopt.IsArg | getopt.Optional, ""},
						},
					},
					"set-build": {
						"set the custom build processes instead of 'go build'",
						getopt.Definitions{
							{"name", "target name", getopt.IsArg | getopt.Required, ""},
							{"cmds", "a command name and arguments", getopt.IsArg | getopt.Required, ""},
						},
					},
				},
			},
			"trace": {
				getopt.Options{
					"control tracer status",
					getopt.Definitions{
						{"command", "command to execute", getopt.IsSubCommand, ""},
					},
				},
				getopt.SubCommands{
					"on": {
						"insert tracing codes to targets",
						getopt.Definitions{
							{"targets", "target names", getopt.IsArg | getopt.Optional, ""},
						},
					},
					"off": {
						"remove tracing codes from targets",
						getopt.Definitions{
							{"targets", "target names", getopt.IsArg | getopt.Optional, ""},
						},
					},
					"status": {
						"show status of tracer",
						getopt.Definitions{
							{"target", "a target name", getopt.IsArg | getopt.Optional, ""},
						},
					},
					"start": {
						"start tracing of running processes. it must be added tracing codes before processes started",
						getopt.Definitions{
							{"targets", "target names", getopt.IsArg | getopt.Optional, ""},
						},
					},
					"stop": {
						"stop tracing of running processes",
						getopt.Definitions{
							{"targets", "target names", getopt.IsArg | getopt.Optional, ""},
						},
					},
				},
			},
			"proc": {
				getopt.Options{
					"build binaries, and start/stop processes",
					getopt.Definitions{
						{"command", "command to execute", getopt.IsSubCommand, ""},
					},
				},
				getopt.SubCommands{
					"build": {
						"build with tracing codes",
						getopt.Definitions{
							{"targets", "target names", getopt.IsArg | getopt.Optional, ""},
						},
					},
					"run": {
						"start processes, and start tracing",
						getopt.Definitions{
							{"targets", "target names", getopt.IsArg | getopt.Optional, ""},
						},
					},
				},
			},
			"log": {
				getopt.Options{
					"manage tracing logs",
					getopt.Definitions{
						{"command", "command to execute", getopt.IsSubCommand, ""},
					},
				},
				getopt.SubCommands{
					"ls": {
						"show log names and histories",
						getopt.Definitions{
							{"target", "a target name", getopt.IsArg | getopt.Optional, ""},
						},
					},
					"show": {
						"show logs on web browser",
						getopt.Definitions{
							{"target", "a target name", getopt.IsArg | getopt.Optional, ""},
						},
					},
				},
			},
		},
	}

	scope, subCommand, options, arguments, passThrough, e := ssco.ParseCommandLine()

	_, wantsHelp := options["help"]

	if e != nil || wantsHelp {
		exit_code := 0

		switch {
		case wantsHelp:
			fmt.Print(ssco.Help())
		default:
			fmt.Printf("%s: %s\nSee '%s --help'.\n",
				info.APP_NAME,
				e.Error(),
				info.APP_NAME)
			exit_code = e.ErrorCode
		}
		return exit_code
	}

	fmt.Printf("scope:\n%s", scope)
	fmt.Printf("subCommand:\n%s", subCommand)
	fmt.Printf("options:\n%#v", options)
	fmt.Printf("arguments: %#v\n", arguments)
	fmt.Printf("passThrough: %#v\n", passThrough)

	conf := config.NewConfig("")
	if err := conf.Load(); err != nil {
		fmt.Println(err)
		return 1
	}

	return 0
}
