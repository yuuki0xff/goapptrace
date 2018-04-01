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
	"os"

	"github.com/spf13/cobra"
	"github.com/yuuki0xff/goapptrace/tracer/encoding"
	"github.com/yuuki0xff/goapptrace/tracer/storage"
	"github.com/yuuki0xff/goapptrace/tracer/types"
)

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use: "validate <FuncLogFile> <GoroutineLog>",
	DisableFlagsInUseLine: true,
	Short: "validate logs",
	Run: func(cmd *cobra.Command, args []string) {
		errlog := log.New(cmd.OutOrStderr(), "ERROR: ", log.Lshortfile)
		stdout := cmd.OutOrStdout()

		var flFile string
		var gFile string
		switch len(args) {
		case 2:
			flFile = args[0]
			gFile = args[1]
		default:
			errlog.Fatalln("invalid args")
		}

		flStore := storage.FuncLogStore{
			Store: storage.Store{
				File:       storage.File(flFile),
				RecordSize: int(encoding.SizeFuncLog()),
				ReadOnly:   true,
			},
		}
		gStore := storage.GoroutineStore{
			Store: storage.Store{
				File:       storage.File(gFile),
				RecordSize: int(encoding.SizeGoroutine()),
				ReadOnly:   true,
			},
		}
		err := flStore.Open()
		if err != nil {
			errlog.Fatalln("Cannot open the FuncLogStore:", err)
		}
		err = gStore.Open()
		if err != nil {
			errlog.Fatalln("Cannot open the GoroutineStore:", err)
		}

		var invalid bool
		fl := types.FuncLogPool.Get().(*types.FuncLog)
		g := &types.Goroutine{}

		flStore.Lock()
		gStore.Lock()
		for gid := int64(0); gid < gStore.Records(); gid++ {
			err := gStore.GetNolock(types.GID(gid), g)
			if err != nil {
				errlog.Fatalf("Cannot get Goroutine(gid=%d): %s\n", gid, err.Error())
			}

			var invalidRow bool
			if g.GID == 0 {
				if gid == 0 {
					// gidが0のときは、全てのフィールドが設定されている or 全てゼロを許容する。
					// それ以外の場合は許容しない。
					if (*g == types.Goroutine{}) {
					} else if g.GID == 0 && g.StartTime != 0 && g.EndTime != 0 {
					} else {
						invalidRow = true
					}
				} else {
					// 全てのフィールドがゼロである必要がある。
					invalidRow = *g != types.Goroutine{}
				}
			} else {
				// 全てのフィールドに何らかの値が設定されている必要がある。
				invalidRow = g.StartTime == 0 || g.EndTime == 0
			}
			if invalidRow {
				invalid = true
				fmt.Fprintf(stdout, "%d: %#v\n", gid, g)
			}
		}

		for id := int64(0); id < flStore.Records(); id++ {
			err = flStore.GetNolock(types.FuncLogID(id), fl)
			if err != nil {
				errlog.Fatalf("Cannot get FuncLog(id=%d): %s\n", id, err.Error())
			}

			invalidRow := fl.ID != types.FuncLogID(id) || fl.StartTime == 0 || fl.EndTime == 0 || len(fl.Frames) == 0

			err = gStore.GetNolock(fl.GID, g)
			if err != nil {
				errlog.Fatalf("Cannot get Goroutine(gid=%d): %s\n", fl.GID, err.Error())
			}
			if fl.IsEnded() {
				invalidRow = invalidRow || fl.GID != g.GID || !(g.StartTime <= fl.StartTime || fl.EndTime <= g.EndTime)
			} else {
				invalidRow = invalidRow || fl.GID != g.GID || !(g.StartTime <= fl.StartTime)
			}

			if invalidRow {
				invalid = true
				fmt.Fprintf(stdout, "%d: %#v\n", id, fl)
			}
		}
		gStore.Unlock()
		flStore.Unlock()

		if invalid {
			os.Exit(1)
		}
	},
}

func init() {
	RootCmd.AddCommand(validateCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// validateCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// validateCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
