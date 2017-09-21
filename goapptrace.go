package main

import (
	"os"

	"log"

	"github.com/yuuki0xff/goapptrace/cmd"
)

func main() {
	os.Exit(realMain())
}

func realMain() int {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	return cmd.Execute()
}
