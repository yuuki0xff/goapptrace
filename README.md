# goapptrace
Goapptrace is a function call tracer for golang.

## Currently State
The goapptrace has not reached a stable release yet.
So interfaces and data formats might be change without notice.

## Installation
```bash
$ go get -u github.com/yuuki0xff/goaptrace
```

## Usage
### 1. Start goapptrace server
The goapptrace consists of server and client. You should start the goapptrace server before tracing.
```bash
$ goapptrace server run &
```
NOTE: The goapptrace server creates `.goapptrace` directory under current working directory, and stores all log files in it.
It might grow to be very large size.

### 2. Start application with goapptrace
If target application can be run with `go run` command, we recommnd using `goapptrace run` command.
```bash
$ goapptrace run -- ./foo.go [args]
```

If you want executable binaries, use `goapptrace build` command instead of `go build`.
When start applications, it requires the `GOAPPTRACE_SERVER` environment variable.
```bash
$ goapptrace build -o ./foo ./file.go
$ GOAPPTRACE_SERVER=tcp://127.0.0.1:8600 ./foo [args]
```

### 3. Show logs
You can see logs with CLI.
```bash
$ goapptrace log ls             # Check LOG_ID.
$ goapptrace log cat "$LOG_ID"  # Print all log messages.
```

### 4. Reduce logs to increase performance
Did your application become unbearably slow down? Are logs too many?
Let's try to disable trace of unnecessary functions.

`goapptrace trace ls` command shows all function names and currently settings.
You can choose functions from it, and change settings by `goapptrace trace start/stop` commands.

```bash
$ goapptrace log ls                # Check LOG_ID.
$ goapptrace trace ls "$LOG_ID"    # Check function names and current status.
$ goapptrace trace stop "$LOG_ID"  # Disable all.
$ goapptrace trace start "$LOG_ID" foo.bar baz.qux
```
Please see `goapptrace --help` for more information about available commands.

## TODO
* Implement TUI completely.
* Collect more information from runtime.ReadTrace().
* Create tests for tracer, builder and CLI commands.
* Create documents.
* Release a stable version.
* Enable linters that are currently disabled.

## Copyright and license
These codes are written by yuuki \<https://github.com/yuuki0xff\>.
Codes released under the MIT license.  
