package cmd

import (
	"bytes"
	"fmt"
	"github.com/mitchellh/cli"
	"github.com/yuuki0xff/goapptrace/info"
	"sort"
	"text/template"
	"strings"
)

var helpTmpl = template.Must(template.New("help").Parse(
`{{.App}} {{if ne .Cmd ""}}{{.Cmd}} {{end}}{{if .HasSubcmd}}<command> {{end}}{{if .HasArgs}}[args]{{end}}

{{ if .HasArgs }}Arguments{{ range $args := .Arguments }}
    {{ $args.NameAligned }}  {{ $args.Synopsis }}{{ end }}

{{ end }}{{ if .HasSubcmd }}Commands{{ range $cmd := .Subcommands }}
    {{ $cmd.NameAligned }}  {{ $cmd.Synopsis }}{{ end }}{{ end }}`))

type helpData struct {
	App         string
	Cmd         string
	HasSubcmd   bool
	HasArgs     bool
	Subcommands []argument
	Arguments   []argument
}
type argument struct {
	NameAligned string
	Synopsis    string
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func maxKeyLength(m map[string]string) int {
	max := 0
	for key := range m {
		l := len(key)
		if max < l {
			max = l
		}
	}
	return max
}

func newArgument(m map[string]string) []argument {
	data := make([]argument, 0, len(m))
	nameMax := maxKeyLength(m)
	format := fmt.Sprintf("%%-%ds", nameMax) // ex: "%-8s"

	for _, k := range sortedKeys(m) {
		v := m[k]
		data = append(data, argument{
			NameAligned: fmt.Sprintf(format, k),
			Synopsis:    v,
		})
	}

	return data
}

func convertSynopsisMap(cfmap map[string]cli.CommandFactory) map[string]string {
	strmap := make(map[string]string)
	for k, cmdFactory := range cfmap {
		c, err := cmdFactory()
		if err != nil {
			panic(err)
		}
		strmap[k] = c.Synopsis()
	}
	return strmap
}

func HelpFunc(cmds []string, subcmds map[string]cli.CommandFactory, args map[string]string) cli.HelpFunc {
	return func(_ map[string]cli.CommandFactory) string {
		return Help(cmds, subcmds, args)
	}
}

func Help(cmds []string, subcmds map[string]cli.CommandFactory, args map[string]string) string {
	var buf bytes.Buffer
	err := helpTmpl.Execute(&buf, helpData{
		App:         info.APP_NAME,
		Cmd:         strings.Join(cmds, " "),
		HasSubcmd:   len(subcmds) > 0,
		HasArgs:     len(args) > 0,
		Subcommands: newArgument(convertSynopsisMap(subcmds)),
		Arguments:   newArgument(args),
	})
	if err != nil {
		panic(err)
	}
	return buf.String()
}

func GetSubCLI(cmd cli.Command, args []string, cmds map[string]cli.CommandFactory) *cli.CLI {
	c := cli.NewCLI(info.APP_NAME, info.VERSION)
	c.Args = args
	c.Commands = cmds
	c.HelpFunc = func(_ map[string]cli.CommandFactory) string {
		return cmd.Help()
	}
	return c
}
