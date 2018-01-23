package cmd

import (
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/yuuki0xff/goapptrace/config"
	"github.com/yuuki0xff/goapptrace/tracer/restapi"
)

func getAPIClient(conf *config.Config) (*restapi.Client, error) {
	if conf == nil || len(conf.Servers.ApiServer) == 0 {
		return nil, errors.New("api server not found")
	}

	var srv *config.ApiServerConfig
	for _, srv = range conf.Servers.ApiServer {
		break
	}
	api := &restapi.Client{
		BaseUrl: srv.Addr,
	}
	if err := api.Init(); err != nil {
		return nil, errors.Wrap(err, "failed to initialize an API client")
	}
	return api, nil
}

// 引数名のprefixを必要に応じて"-"から"--"に変換してから、再実行する。
// golang標準のflagパッケージの形式との互換性を持たせるために使用する。
func fixFlagName(command *cobra.Command, e error) error {
	// 定義済みの長いフラグ名 ("--"は含まない)
	flagNames := map[string]bool{}
	command.Flags().VisitAll(func(flag *pflag.Flag) {
		flagNames[flag.Name] = true
	})

	var converted bool
	args := []string{}
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-") && flagNames[arg[1:]] {
			// "-flag"から"--flag"形式に変換する。
			args = append(args, "--"+arg[1:])
			converted = true
		} else {
			args = append(args, arg)
		}
	}

	if !converted {
		// 引数の変換が行えないにも関わらずエラーが発生した状況である。
		// 間違った引数を与えていた可能性があるので、ここで実行を中断。
		return e
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}

	cmd := exec.Command(exe, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
