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
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/yuuki0xff/goapptrace/config"
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
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
	api, err := getAPIClient(conf)
	if err != nil {
		return err
	}

	ch, err := api.SearchFuncCalls(logID, restapi.SearchFuncCallParams{
		SortKey:   restapi.SortByStartTime,
		SortOrder: restapi.AscendingSortOrder,
		//Limit:     1000,
	})
	if err != nil {
		return err
	}
	defer func() {
		// consume all items.
		for range ch {
		}
	}()

	logw.SetFuncStatusInfoGetter(func(id logutil.FuncStatusID) restapi.FuncStatusInfo {
		s, err := api.FuncStatus(logID, strconv.Itoa(int(id)))
		if err != nil {
			log.Panic(err)
		}
		return s
	})
	logw.SetFuncInfoGetter(func(id logutil.FuncID) restapi.FuncInfo {
		f, err := api.Func(logID, strconv.Itoa(int(id)))
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
	SetFuncInfoGetter(func(id logutil.FuncID) restapi.FuncInfo)
	SetFuncStatusInfoGetter(func(id logutil.FuncStatusID) restapi.FuncStatusInfo)
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

func (w *JsonLogWriter) SetFuncInfoGetter(func(id logutil.FuncID) restapi.FuncInfo) {
}
func (w *JsonLogWriter) SetFuncStatusInfoGetter(func(id logutil.FuncStatusID) restapi.FuncStatusInfo) {
}

type TextLogWriter struct {
	output     io.Writer
	funcInfo   func(id logutil.FuncID) restapi.FuncInfo
	funcStatus func(id logutil.FuncStatusID) restapi.FuncStatusInfo
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
	fs := w.funcStatus(currentFrame)
	funcName := w.funcInfo(fs.Func).Name // module.func
	line := fs.Line
	execTime := evt.EndTime - evt.StartTime

	_, err := fmt.Fprintf(
		w.output,
		"%s %d [%d] %s:%d\n",
		evt.StartTime.UnixTime().Format(config.TimestampFormat),
		execTime,
		evt.GID,
		funcName, // module.func
		line,     // line
	)
	return err
}
func (w *TextLogWriter) SetFuncInfoGetter(f func(id logutil.FuncID) restapi.FuncInfo) {
	w.funcInfo = f
}
func (w *TextLogWriter) SetFuncStatusInfoGetter(f func(id logutil.FuncStatusID) restapi.FuncStatusInfo) {
	w.funcStatus = f
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
