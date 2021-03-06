// Copyright © 2018 yuuki0xff <yuuki0xff@gmail.com>
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
	"fmt"
	"log"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/yuuki0xff/goapptrace/tracer/storage"
)

// dumpIndexCmd represents the dumpIndex command
var dumpIndexCmd = &cobra.Command{
	Use: "index <file> <id>",
	DisableFlagsInUseLine: true,
	Short: "dump an index file",
	Run: func(cmd *cobra.Command, args []string) {
		errlog := log.New(cmd.OutOrStderr(), "ERROR: ", log.Lshortfile)
		stdout := cmd.OutOrStdout()

		var fpath string
		var strId string

		switch len(args) {
		case 2:
			strId = args[1]
			fallthrough
		case 1:
			fpath = args[0]
		default:
			errlog.Fatalln("invalid args")
		}

		index := storage.Index{
			File:     storage.File(fpath),
			ReadOnly: true,
		}
		err := index.Open()
		if err != nil {
			errlog.Fatalln("Cannot open index file:", err)
		}

		err = index.Load()
		if err != nil {
			errlog.Fatalln("Cannot load from index file:", err)
		}

		if strId == "" {
			fmt.Fprintln(stdout, "records:", index.Len())
			for id := int64(0); id < index.Len(); id++ {
				ir := index.Get(id)
				fmt.Fprintf(stdout, "%d: %#v\n", id, ir)
			}
		} else {
			id, err := strconv.ParseInt(strId, 10, 64)
			if err != nil {
				errlog.Fatalln("Invalid ID:")
			}
			ir := index.Get(id)
			fmt.Fprintf(stdout, "%d: %#v\n", id, ir)
		}
	},
}

func init() {
	dumpCmd.AddCommand(dumpIndexCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// dumpIndexCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// dumpIndexCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
