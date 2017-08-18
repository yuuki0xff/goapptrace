package main

import (
	"github.com/mitchellh/cli"
	"github.com/yuuki0xff/goapptrace/cmd"
	"github.com/yuuki0xff/goapptrace/info"
	"log"
	"os"
	"flag"
)

func main() {
	os.Exit(realMain())
}

func realMain() int {
	flag.Parse()
	c := cli.NewCLI(info.APP_NAME, info.VERSION)
	c.Args = os.Args[1:]
	c.Commands = cmd.Commands
	c.HelpFunc = cmd.HelpFunc([]string{}, cmd.Commands, map[string]string{})
	c.IsHelp()

	exit, err := c.Run()
	if err != nil {
		log.Println(err)
	}
	return exit
}
