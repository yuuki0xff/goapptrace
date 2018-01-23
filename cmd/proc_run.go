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
	"fmt"
	"io"
	"log"
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
	RunE: wrap(func(conf *config.Config, cmd *cobra.Command, args []string) error {
		conf.WantSave()
		return runProcRun(conf, cmd.OutOrStdout(), cmd.OutOrStderr(), args)
	}),
}

func runProcRun(conf *config.Config, stdout, stderr io.Writer, targets []string) error {
	if len(targets) == 0 {
		targets = conf.Targets.Names()
	}

	srv, err := getLogServer(conf)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
	}

	env := procRunEnv(srv)

	var lastErr error
	wg := sync.WaitGroup{}
	for _, targetName := range targets {
		target, err := conf.Targets.Get(config.TargetName(targetName))
		if err != nil {
			return err
		}
		proc, err := target.Run.Start(env)
		if err != nil {
			return err
		}
		wg.Add(1)
		go func() {
			if err := proc.Wait(); err != nil {
				wrapped := fmt.Errorf("failed run a command (%s): %s", proc.Args, err.Error())
				lastErr = wrapped
				log.Printf("WARN: %s", wrapped)
			}
			wg.Done()
		}()
	}

	wg.Wait()
	return lastErr
}

// トレース対象のプロセスの環境変数を返す
func procRunEnv(srv restapi.ServerStatus) []string {
	env := os.Environ()
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
