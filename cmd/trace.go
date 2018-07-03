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
	"regexp"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/yuuki0xff/goapptrace/tracer/restapi"
	"github.com/yuuki0xff/goapptrace/tracer/types"
)

// traceCmd represents the trace command
var traceCmd = &cobra.Command{
	Use:   "trace",
	Short: "Manage tracer status",
}

func init() {
	RootCmd.AddCommand(traceCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// traceCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// traceCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

type tracePattern struct {
	LogID    string
	Api      restapi.ClientWithCtx
	Names    []string
	IsRegexp bool //*regexp.Regexp
	Opt      *handlerOpt
}

// トレース対象となる関数名を返す。
func (t *tracePattern) SymbolNames() ([]string, error) {
	var symbolNames []string
	if t.IsRegexp || len(t.Names) == 0 {
		symbols, err := t.Api.Symbols(t.LogID)
		if err != nil {
			return nil, err
		}
		err = symbols.Save(func(data types.SymbolsData) error {
			for _, f := range data.Funcs {
				symbolNames = append(symbolNames, f.Name)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		symbolNames = t.Names
	}
	return symbolNames, nil
}

// シンボル名が、 tracePattern.Names にマッチするか評価する関数を返す。
func (t *tracePattern) Matcher() (func(s string) bool, error) {
	if t.IsRegexp {
		// 正規表現をコンパイル
		var regs []*regexp.Regexp
		for _, expr := range t.Names {
			reg, err := regexp.Compile(expr)
			if err != nil {
				return nil, errors.Wrapf(err, "Invalid regular expression: \"%s\"", expr)
			}
			regs = append(regs, reg)
		}

		return func(s string) bool {
			for _, reg := range regs {
				if reg.FindStringIndex(s) != nil {
					return true
				}
			}
			return false
		}, nil
	} else {
		return func(s string) bool {
			for _, name := range t.Names {
				if s == name {
					return true
				}
			}
			return false
		}, nil
	}
}
