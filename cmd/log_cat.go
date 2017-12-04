// Copyright © 2017 yuuki0xff <yuuki0xff@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"log"

	"fmt"

	"github.com/spf13/cobra"
	"github.com/yuuki0xff/goapptrace/config"
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
	"github.com/yuuki0xff/goapptrace/tracer/storage"
)

// logCatCmd represents the cat command
var logCatCmd = &cobra.Command{
	Use:   "cat",
	Short: "Show logs on console",
	RunE: wrap(func(conf *config.Config, cmd *cobra.Command, args []string) error {
		stderr := cmd.OutOrStderr()

		strg := &storage.Storage{
			Root: storage.DirLayout{Root: conf.LogsDir()},
		}
		if err := strg.Init(); err != nil {
			fmt.Fprintf(stderr, "Failed Storage.Init(): %s", err.Error())
		}

		if len(args) != 1 {
			fmt.Fprintf(stderr, "Should specify one args")
		}
		logID := storage.LogID{}
		logID, err := logID.Unhex(args[0])
		if err != nil {
			fmt.Fprintf(stderr, "Invalid LogID: %s", err.Error())
		}

		if err := runLogCat(strg, logID); err != nil {
			fmt.Fprint(stderr, err)
		}
		return nil
	}),
}

func runLogCat(strg *storage.Storage, id storage.LogID) error {
	logobj, ok := strg.Log(id)
	if !ok {
		return fmt.Errorf("LogID(%s) not found", id.Hex())
	}

	var i int
	if err := logobj.Walk(func(evt logutil.RawFuncLogNew) error {
		log.Printf("%d: %+v", i, evt)
		i++
		return nil
	}); err != nil {
		return fmt.Errorf("log read error: %s", err)
	}
	return nil
}

func init() {
	logCmd.AddCommand(logCatCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// logCatCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// logCatCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
