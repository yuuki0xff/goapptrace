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

	"encoding/json"

	"time"

	"github.com/spf13/cobra"
	"github.com/yuuki0xff/goapptrace/config"
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
	"github.com/yuuki0xff/goapptrace/tracer/storage"
)

const (
	DefaultTimeFormat = "15:04:05.000"
)

// logCatCmd represents the cat command
var logCatCmd = &cobra.Command{
	Use:   "cat",
	Short: "Show logs on console",
	RunE: wrap(func(conf *config.Config, cmd *cobra.Command, args []string) error {
		stdout := cmd.OutOrStdout()

		strg := &storage.Storage{
			Root:     storage.DirLayout{Root: conf.LogsDir()},
			ReadOnly: true,
		}
		if err := strg.Init(); err != nil {
			return fmt.Errorf("Failed Storage.Init(): %s", err.Error())
		}

		// get specify log object.
		if len(args) != 1 {
			return fmt.Errorf("Should specify one args")
		}
		logID, err := storage.LogID{}.Unhex(args[0])
		if err != nil {
			return fmt.Errorf("Invalid LogID: %s", err.Error())
		}
		logobj, ok := strg.Log(logID)
		if !ok {
			return fmt.Errorf("LogID(%s) not found", logID.Hex())
		}
		defer logobj.Close() // nolinter: errchk

		// initialize LogWriter with specify format.
		format, err := cmd.Flags().GetString("format")
		if err != nil {
			return fmt.Errorf("Flag error: %s", err.Error())
		}
		writer, err := NewLogWriter(format, stdout)
		if err != nil {
			return fmt.Errorf("failed to initialize LogWriter(%s): %s", logID, err.Error())
		}
		return runLogCat(logobj, writer)
	}),
}

func runLogCat(logobj *storage.Log, writer LogWriter) error {
	writer.SetSymbols(logobj.Symbols())

	if err := writer.WriteHeader(); err != nil {
		return err
	}
	if err := logobj.WalkRawFuncLog(func(evt logutil.RawFuncLog) error {
		return writer.Write(evt)
	}); err != nil {
		return fmt.Errorf("log read error: %s", err)
	}
	return nil
}

type LogWriter interface {
	WriteHeader() error
	Write(evt logutil.RawFuncLog) error
	SetSymbols(symbols *logutil.Symbols)
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
func (w *JsonLogWriter) Write(evt logutil.RawFuncLog) error {
	return w.encoder.Encode(evt)
}
func (w *JsonLogWriter) SetSymbols(symbols *logutil.Symbols) {}

type TextLogWriter struct {
	output  io.Writer
	symbols *logutil.Symbols
}

func NewTextLogWriter(output io.Writer) *TextLogWriter {
	return &TextLogWriter{
		output: output,
	}
}
func (w *TextLogWriter) WriteHeader() error {
	_, err := fmt.Fprintln(w.output, "[Tag] Timestamp ExecTime GID TxID Module.Func:Line")
	return err
}
func (w *TextLogWriter) Write(evt logutil.RawFuncLog) error {
	currentFrame := evt.Frames[0]
	fs := w.symbols.FuncStatus[currentFrame]
	funcName := w.symbols.Funcs[fs.Func].Name // module.func
	line := fs.Line
	execTime := 0 // TODO: calc ExecTime

	_, err := fmt.Fprintf(
		w.output,
		"[%s] %s %d %d %d %s:%d\n",
		evt.Tag,
		time.Unix(evt.Timestamp, 0).Format(DefaultTimeFormat),
		execTime,
		evt.GID,
		evt.TxID,
		funcName, // module.func
		line,     // line
	)
	return err
}
func (w *TextLogWriter) SetSymbols(symbols *logutil.Symbols) {
	w.symbols = symbols
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
