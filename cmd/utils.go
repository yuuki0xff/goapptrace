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

// sharedFlags are shared by the "build" and "run" commands.
var sharedFlags = pflag.NewFlagSet("", pflag.ContinueOnError)

func init() {
	sharedFlags.BoolP("a", "a", false, "force rebuilding of packages that are already up-to-date.")
	sharedFlags.BoolP("n", "n", false, "print the commands but do not run them.")
	sharedFlags.IntP("p", "p", 0, "specifies the number of threads/commands to run.")
	sharedFlags.BoolP("race", "", false, "enable data race detection.")
	sharedFlags.BoolP("msan", "", false, "enable interoperation with memory sanitizer.")
	sharedFlags.BoolP("v", "v", false, "print the names of packages as they are compiled.")
	sharedFlags.BoolP("work", "", false, "print the name of the temporary work directory and do not delete it when exiting.")
	sharedFlags.BoolP("x", "x", false, "print the commands.")
	sharedFlags.StringP("asmflags", "", "", "arguments to pass on each go tool asm invocation.")
	sharedFlags.StringP("buildmode", "", "", "build mode to use. See 'go help buildmode' for more.")
	sharedFlags.StringP("compiler", "", "", "name of compiler to use, as in runtime.Compiler (gccgo or gc).")
	sharedFlags.StringP("gccgoflags", "", "", "arguments to pass on each gccgo compiler/linker invocation.")
	sharedFlags.StringP("gcflags", "", "", "arguments to pass on each go tool compile invocation.")
	sharedFlags.StringP("installsuffix", "", "", "a suffix to use in the name of the package installation directory.")
	sharedFlags.StringP("ldflags", "", "", "arguments to pass on each go tool link invocation.")
	sharedFlags.BoolP("linkshared", "", false, "link against shared libraries previously created with -buildmode=shared.")
	sharedFlags.StringP("pkgdir", "", "", "install and load all packages from dir instead of the usual locations.")
	sharedFlags.StringP("tags", "", "", "a space-separated list of build tags to consider satisfied during the build.")
	sharedFlags.StringP("toolexec", "", "", "a program to use to invoke toolchain programs like vet and asm.")
}

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
