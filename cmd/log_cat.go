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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/yuuki0xff/goapptrace/config"
	"github.com/yuuki0xff/goapptrace/tracer/restapi"
	"github.com/yuuki0xff/goapptrace/tracer/types"
)

// logCatCmd represents the cat command
var logCatCmd = &cobra.Command{
	Use: "cat <id>",
	DisableFlagsInUseLine: true,
	Short: "Show logs on console",
	RunE:  wrap(runLogCatfunc),
}

func runLogCatfunc(opt *handlerOpt) error {
	// get specify log object.
	if len(opt.Args) != 1 {
		opt.ErrLog.Println("Should specify one args")
		return errInvalidArgs
	}
	logID := opt.Args[0]

	format, err := opt.Cmd.Flags().GetString("format")
	if err != nil {
		opt.ErrLog.Println("Invalid format:", err)
		return errInvalidArgs
	}
	api, cancel, err := opt.ApiWithCancel(context.Background())
	if err != nil {
		opt.ErrLog.Println(err)
		return errInvalidArgs
	}

	writer, err := NewLogWriter(format, opt.Stdout)
	if err != nil {
		opt.ErrLog.Printf("Failed to initialize LogWriter(%s): %s\n", logID, err)
		return errGeneral
	}

	ch, eg := api.SearchFuncLogs(logID, restapi.SearchFuncLogParams{
		SortKey:   restapi.SortByStartTime,
		SortOrder: restapi.AscendingSortOrder,
		//Limit:     1000,
	})
	eg.Go(func() error {
		defer cancel()
		writer.SetGoLineGetter(func(pc uintptr) types.GoLine {
			s, err := api.GoLine(logID, pc)
			if err != nil {
				cancel()
				opt.ErrLog.Panic(err)
			}
			return s
		})
		writer.SetGoFuncGetter(func(pc uintptr) types.GoFunc {
			f, err := api.GoFunc(logID, pc)
			if err != nil {
				cancel()
				opt.ErrLog.Panic(err)
			}
			return f
		})
		if err := writer.WriteHeader(); err != nil {
			opt.ErrLog.Println(err)
			return errIo
		}
		for funcCall := range ch {
			if err := writer.Write(funcCall); err != nil {
				opt.ErrLog.Println(err)
				return errIo
			}
		}
		return nil
	})
	err = eg.Wait()
	if err != nil {
		if err == errIo {
			return errIo
		}
		opt.ErrLog.Println("ERROR: Received unexpected response:", err)
		return errGeneral
	}
	return nil
}

type LogWriter interface {
	WriteHeader() error
	Write(evt types.FuncLog) error
	SetGoFuncGetter(func(pc uintptr) types.GoFunc)
	SetGoLineGetter(func(pc uintptr) types.GoLine)
}

func NewLogWriter(format string, out io.Writer) (LogWriter, error) {
	var writer LogWriter
	switch format {
	case "json":
		writer = NewJsonLogWriter(out)
	case "text":
		fallthrough
	case "":
		writer = NewTextLogWriter(out)
	default:
		return nil, fmt.Errorf("Invalid format: %s", format)
	}
	return writer, nil
}

type JsonLogWriter struct {
	encoder *json.Encoder
}

func NewJsonLogWriter(output io.Writer) *JsonLogWriter {
	encoder := json.NewEncoder(output)
	encoder.SetEscapeHTML(false)

	return &JsonLogWriter{
		encoder: encoder,
	}
}
func (w *JsonLogWriter) WriteHeader() error {
	return nil
}
func (w *JsonLogWriter) Write(evt types.FuncLog) error {
	return w.encoder.Encode(evt)
}

func (w *JsonLogWriter) SetGoFuncGetter(func(pc uintptr) types.GoFunc) {
}
func (w *JsonLogWriter) SetGoLineGetter(func(pc uintptr) types.GoLine) {
}

type TextLogWriter struct {
	output   io.Writer
	funcInfo func(pc uintptr) types.GoFunc
	goLine   func(pc uintptr) types.GoLine
}

func NewTextLogWriter(output io.Writer) *TextLogWriter {
	return &TextLogWriter{
		output: output,
	}
}
func (w *TextLogWriter) WriteHeader() error {
	_, err := fmt.Fprintln(w.output, "StartTime ExecTime [GID] Module.Func:Line")
	return err
}
func (w *TextLogWriter) Write(evt types.FuncLog) error {
	currentFrame := evt.Frames[0]
	fs := w.goLine(currentFrame)
	funcName := w.funcInfo(currentFrame).Name // module.func
	line := fs.Line

	// 実行が終了していない場合、実行時間は"*"と表示する
	execTime := "*"
	if evt.IsEnded() {
		if evt.EndTime-evt.StartTime < 0 {
			// validation error
			log.Panicf("negative execTime: evt=%+v", evt)
		}
		execTime = strconv.FormatInt(int64(evt.EndTime-evt.StartTime), 10)
	}

	_, err := fmt.Fprintf(
		w.output,
		"%s %s [%d] %s:%d\n",
		evt.StartTime.UnixTime().Format(config.TimestampFormat),
		execTime,
		evt.GID,
		funcName, // module.func
		line,     // line
	)
	return err
}
func (w *TextLogWriter) SetGoFuncGetter(f func(pc uintptr) types.GoFunc) {
	w.funcInfo = f
}
func (w *TextLogWriter) SetGoLineGetter(f func(pc uintptr) types.GoLine) {
	w.goLine = f
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
	logCatCmd.Flags().StringP("format", "f", "text", `Specify output format. You can choose "json" or "text"`)
}
