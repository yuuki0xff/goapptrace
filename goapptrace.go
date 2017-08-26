package main

import (
	"os"

	"github.com/yuuki0xff/goapptrace/cmd"
)

func main() {
	os.Exit(realMain())
}

func realMain() int {
	return cmd.Execute()
}
