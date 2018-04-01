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
	"github.com/yuuki0xff/goapptrace/tracer/encoding"
	"github.com/yuuki0xff/goapptrace/tracer/storage"
	"github.com/yuuki0xff/goapptrace/tracer/types"
)

// dumpFuncCmd represents the dumpFunc command
var dumpFuncCmd = &cobra.Command{
	Use: "func <file> [FuncLogID]",
	DisableFlagsInUseLine: true,
	Short: "dumps function call logs",
	Run: func(cmd *cobra.Command, args []string) {
		errlog := log.New(cmd.OutOrStderr(), "ERROR: ", log.Lshortfile)
		stdout := cmd.OutOrStdout()

		var fpath string
		var flid string

		switch len(args) {
		case 2:
			flid = args[1]
			fallthrough
		case 1:
			fpath = args[0]
		default:
			errlog.Fatalln("invalid args")
		}

		store := storage.FuncLogStore{
			Store: storage.Store{
				File:       storage.File(fpath),
				RecordSize: int(encoding.SizeFuncLog()),
				ReadOnly:   true,
			},
		}
		err := store.Open()
		if err != nil {
			errlog.Fatalln("Cannot open the FuncLogStore:", err)
		}

		fl := types.FuncLogPool.Get().(*types.FuncLog)
		if flid == "" {
			fmt.Fprintln(stdout, "records:", store.Records())
			store.Lock()
			for id := int64(0); id < store.Records(); id++ {
				err := store.GetNolock(types.FuncLogID(id), fl)
				if err != nil {
					errlog.Fatalf("Cannot get FuncLog(id=%d): %s\n", id, err.Error())
				}
				fmt.Fprintf(stdout, "%d: %#v\n", id, fl)
			}
			store.Unlock()
		} else {
			id, err := strconv.ParseInt(flid, 10, 64)
			if err != nil {
				errlog.Fatalln("Invalid FuncLogID:", err)
			}

			err = store.Get(types.FuncLogID(id), fl)
			if err != nil {
				errlog.Fatalf("Cannot get FuncLog(id=%d): %s\n", id, err.Error())
			}
			fmt.Fprintf(stdout, "%d: %#v\n", id, fl)
		}
	},
}

func init() {
	dumpCmd.AddCommand(dumpFuncCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// dumpFuncCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// dumpFuncCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
