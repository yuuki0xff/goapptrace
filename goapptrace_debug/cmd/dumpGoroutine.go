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
	"github.com/yuuki0xff/goapptrace/tracer/types"
)

// dumpGoroutineCmd represents the dumpGoroutine command
var dumpGoroutineCmd = &cobra.Command{
	Use: "goroutine <file> [GID]",
	DisableFlagsInUseLine: true,
	Short: "dumps goroutine information",
	Run: func(cmd *cobra.Command, args []string) {
		errlog := log.New(cmd.OutOrStderr(), "ERROR: ", log.Lshortfile)
		stdout := cmd.OutOrStdout()

		var fpath string
		var strGid string

		switch len(args) {
		case 2:
			strGid = args[1]
			fallthrough
		case 1:
			fpath = args[0]
		default:
			errlog.Fatalln("invalid args")
		}

		store := storage.GoroutineStore{
			Store: storage.Store{
				File:       storage.File(fpath),
				RecordSize: 128, //int(encoding.SizeGoroutine()),
				ReadOnly:   true,
			},
		}
		err := store.Open()
		if err != nil {
			errlog.Fatalln("Cannot open the GoroutineStore:", err)
		}

		g := &types.Goroutine{}
		if strGid == "" {
			fmt.Fprintln(stdout, "records:", store.Records())
			store.Lock()
			for gid := int64(0); gid < store.Records(); gid++ {
				err := store.GetNolock(types.GID(gid), g)
				if err != nil {
					errlog.Fatalf("Cannot get Goroutine(gid=%d): %s\n", gid, err.Error())
				}
				fmt.Fprintf(stdout, "%d: %#v\n", gid, g)
			}
			store.Unlock()
		} else {
			gid, err := strconv.ParseInt(strGid, 10, 64)
			if err != nil {
				errlog.Fatalln("Invalid GID:", err)
			}

			err = store.Get(types.GID(gid), g)
			if err != nil {
				errlog.Fatalf("Cannot get Goroutine(id=%d): %s\n", gid, err.Error())
			}
			fmt.Fprintf(stdout, "%d: %#v\n", gid, g)
		}
	},
}

func init() {
	dumpCmd.AddCommand(dumpGoroutineCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// dumpGoroutineCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// dumpGoroutineCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
