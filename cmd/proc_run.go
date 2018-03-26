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
	"os"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/yuuki0xff/goapptrace/config"
	"github.com/yuuki0xff/goapptrace/info"
	"github.com/yuuki0xff/goapptrace/tracer/restapi"
)

const (
	CloseDelayDuration = 3 * time.Second
)

// procRunCmd represents the run command
var procRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Start processes, and start tracing",
	RunE:  wrap(runProcRun),
}

func runProcRun(opt *handlerOpt) error {
	opt.Conf.WantSave()
	targets := opt.Args
	if len(targets) == 0 {
		targets = opt.Conf.Targets.Names()
	}

	srv, err := opt.LogServer()
	if err != nil {
		opt.ErrLog.Println(err)
		return errGeneral
	}
	env := append(os.Environ(), procRunEnv(srv)...)

	var lastErr error
	wg := sync.WaitGroup{}
	for _, targetName := range targets {
		target, err := opt.Conf.Targets.Get(config.TargetName(targetName))
		if err != nil {
			opt.ErrLog.Println(err)
			return errGeneral
		}
		proc, err := target.Run.Start(env)
		if err != nil {
			opt.ErrLog.Println(err)
			return errGeneral
		}
		wg.Add(1)
		go func() {
			if err := proc.Wait(); err != nil {
				opt.ErrLog.Printf("WARN: Failed run a command (%+v): %s\n", proc.Args, err.Error())
				return
			}
			wg.Done()
		}()
	}

	wg.Wait()
	return lastErr
}

// トレース対象のプロセスの環境変数を返す
func procRunEnv(srv restapi.ServerStatus) (env []string) {
	env = append(env, info.DEFAULT_LOGSRV_ENV+"="+srv.Addr)
	return env
}

func init() {
	procCmd.AddCommand(procRunCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// procRunCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// procRunCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
