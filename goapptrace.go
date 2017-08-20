package main

import (
	"github.com/yuuki0xff/goapptrace/cmd"
	"os"
)

func main() {
	os.Exit(realMain())
}

func realMain() int {
	return cmd.Execute()
}
