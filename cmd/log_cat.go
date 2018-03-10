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
)

// logCatCmd represents the cat command
var logCatCmd = &cobra.Command{
	Use:   "cat",
	Short: "Show logs on console",
	RunE: wrap(func(conf *config.Config, cmd *cobra.Command, args []string) error {
		// get specify log object.
		if len(args) != 1 {
			return fmt.Errorf("Should specify one args")
		}
		logID := args[0]

		// initialize LogWriter with specify format.
		format, err := cmd.Flags().GetString("format")
		if err != nil {
			return fmt.Errorf("Flag error: %s", err.Error())
		}
		writer, err := NewLogWriter(format, cmd.OutOrStdout())
		if err != nil {
			return fmt.Errorf("failed to initialize LogWriter(%s): %s", logID, err.Error())
		}

		return runLogCat(conf, cmd.OutOrStderr(), logID, writer)
	}),
}

func runLogCat(conf *config.Config, stderr io.Writer, logID string, logw LogWriter) error {
	// TODO: timeoutに対応させる
	apiNoctx, err := getAPIClient(conf)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	api := apiNoctx.WithCtx(ctx)

	ch, err := api.SearchFuncCalls(logID, restapi.SearchFuncCallParams{
		SortKey:   restapi.SortByStartTime,
		SortOrder: restapi.AscendingSortOrder,
		//Limit:     1000,
	})
	if err != nil {
		return err
	}
	defer func() {
		cancel()
		// consume all items.
		for range ch {
		}
	}()

	logw.SetGoLineInfoGetter(func(pc uintptr) restapi.GoLineInfo {
		s, err := api.GoLine(logID, strconv.Itoa(int(pc)))
		if err != nil {
			log.Panic(err)
		}
		return s
	})
	logw.SetFuncInfoGetter(func(pc uintptr) restapi.FuncInfo {
		f, err := api.Func(logID, strconv.Itoa(int(pc)))
		if err != nil {
			log.Panic(err)
		}
		return f
	})
	if err := logw.WriteHeader(); err != nil {
		return err
	}
	for funcCall := range ch {
		if err := logw.Write(funcCall); err != nil {
			return err
		}
	}
	return nil
}

type LogWriter interface {
	WriteHeader() error
	Write(evt restapi.FuncCall) error
	SetFuncInfoGetter(func(pc uintptr) restapi.FuncInfo)
	SetGoLineInfoGetter(func(pc uintptr) restapi.GoLineInfo)
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
func (w *JsonLogWriter) Write(evt restapi.FuncCall) error {
	return w.encoder.Encode(evt)
}

func (w *JsonLogWriter) SetFuncInfoGetter(func(pc uintptr) restapi.FuncInfo) {
}
func (w *JsonLogWriter) SetGoLineInfoGetter(func(pc uintptr) restapi.GoLineInfo) {
}

type TextLogWriter struct {
	output   io.Writer
	funcInfo func(pc uintptr) restapi.FuncInfo
	goLine   func(pc uintptr) restapi.GoLineInfo
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
func (w *TextLogWriter) Write(evt restapi.FuncCall) error {
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
func (w *TextLogWriter) SetFuncInfoGetter(f func(pc uintptr) restapi.FuncInfo) {
	w.funcInfo = f
}
func (w *TextLogWriter) SetGoLineInfoGetter(f func(pc uintptr) restapi.GoLineInfo) {
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
