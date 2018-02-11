package cmd

import (
	"context"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/yuuki0xff/goapptrace/config"
	"github.com/yuuki0xff/goapptrace/tracer/builder"
	"github.com/yuuki0xff/goapptrace/tracer/restapi"
)

// sharedFlags are shared by the "build" and "run" commands.
func sharedFlags() *pflag.FlagSet {
	f := pflag.NewFlagSet("", pflag.ContinueOnError)
	f.BoolP("a", "", false, "force rebuilding of packages that are already up-to-date.")
	f.BoolP("n", "", false, "print the commands but do not run them.")
	f.IntP("p", "", 0, "specifies the number of threads/commands to run.")
	f.BoolP("race", "", false, "enable data race detection.")
	f.BoolP("msan", "", false, "enable interoperation with memory sanitizer.")
	f.BoolP("v", "", false, "print the names of packages as they are compiled.")
	f.BoolP("work", "", false, "print the name of the temporary work directory and do not delete it when exiting.")
	f.BoolP("x", "", false, "print the commands.")
	f.StringP("asmflags", "", "", "arguments to pass on each go tool asm invocation.")
	f.StringP("buildmode", "", "", "build mode to use. See 'go help buildmode' for more.")
	f.StringP("compiler", "", "", "name of compiler to use, as in runtime.Compiler (gccgo or gc).")
	f.StringP("gccgoflags", "", "", "arguments to pass on each gccgo compiler/linker invocation.")
	f.StringP("gcflags", "", "", "arguments to pass on each go tool compile invocation.")
	f.StringP("installsuffix", "", "", "a suffix to use in the name of the package installation directory.")
	f.StringP("ldflags", "", "", "arguments to pass on each go tool link invocation.")
	f.BoolP("linkshared", "", false, "link against f libraries previously created with -buildmode=f.")
	f.StringP("pkgdir", "", "", "install and load all packages from dir instead of the usual locations.")
	f.StringP("tags", "", "", "a space-separated list of build tags to consider satisfied during the build.")
	f.StringP("toolexec", "", "", "a program to use to invoke toolchain programs like vet and asm.")
	return f
}

func sharedFlagNames() map[string]bool {
	names := map[string]bool{}
	sharedFlags().VisitAll(func(flag *pflag.Flag) {
		names[flag.Name] = true
	})
	return names
}

func mergeFlagNames(a, b map[string]bool) map[string]bool {
	for key, value := range b {
		a[key] = value
	}
	return a
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
func fixFlagName(flagNames map[string]bool) func(command *cobra.Command, e error) error {
	return func(command *cobra.Command, e error) error {
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

		// ignore an error of "Subprocess launching with variable" because arguments are specified by the trusted user.
		cmd := exec.Command(exe, args...) // nolint: gas
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
}

// FlagSetからgolang標準のflagパッケージが解釈可能な形式の引数へと変換する。
func toShortPrefixFlag(flagset *pflag.FlagSet, flags map[string]bool) []string {
	args := []string{}
	flagset.Visit(func(flag *pflag.Flag) {
		if !flags[flag.Name] {
			return
		}
		flagname := "-" + flag.Name

		value := flag.Value.String()
		switch flag.Value.Type() {
		case "bool":
			if value == "true" {
				args = append(args, flagname)
			}
		case "string":
			if value != "" {
				args = append(args, flagname, value)
			}
		case "int":
			if value != "0" {
				args = append(args, flagname, value)
			}
		default:
			log.Panicf("invalid type name: %s", flag.Value.Type())
		}
	})
	return args
}

func prepareRepo(tmpdir string, targets []string, conf *config.Config) (*builder.RepoBuilder, error) {
	goroot := path.Join(tmpdir, "goroot")
	gopath := path.Join(tmpdir, "gopath")

	ignoreFiles := map[string]bool{}
	if err := conf.Targets.Walk(nil, func(t *config.Target) error {
		for _, f := range t.Files {
			if trace, ok := t.Trace[f]; ok {
				if !trace.IsTracing {
					ignoreFiles[f] = true
				}
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	b := &builder.RepoBuilder{
		OrigGopath: os.Getenv("GOPATH"),
		Goroot:     goroot,
		Gopath:     gopath,
		IgnorePkgs: map[string]bool{
			"github.com/yuuki0xff/goapptrace/tracer/logger": true,
		},
		IgnoreFiles:   ignoreFiles,
		IgnoreStdPkgs: true,
	}
	if err := b.Init(); err != nil {
		return nil, err
	}

	// insert logging codes
	if err := b.EditAll(targets); err != nil {
		return nil, err
	}
	return b, nil
}

func getLogServer(conf *config.Config) (srv restapi.ServerStatus, err error) {
	apiNoctx, err := getAPIClient(conf)
	if err != nil {
		return
	}
	// TODO: ctxを外部から渡す。
	api := apiNoctx.WithCtx(context.Background())
	srvs, err := api.Servers()
	if err != nil {
		return
	}
	for _, srv = range srvs {
		return
	}
	return srv, errors.New("log servers is not running")
}
