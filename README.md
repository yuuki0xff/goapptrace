# goapptrace
Goapptrace is a function call tracer for golang.

## Currently State
The goapptrace has not reached a stable release yet.
So Interfaces and data formats may be changed without notice.

## Installation
```bash
$ go get -u github.com/yuuki0xff/goaptrace
```

## Usage
### 1. Start goapptrace server
```bash
$ goapptrace server run &
```

### 2. Start application with goapptracer
If target application can be run with "go run" command, we recommnd using "goapptrace run" command.
```bash
$ goapptrace run -- ./foo.go
```

### 3. Show logs
You can see logs with TUI (Text-based User Interface).
```bash
$ goapptrace tui
```

In addition to TUI, CLI commands are available.
```bash
$ goapptrace log ls
$ goapptrace log cat "$LOG_ID"
```
Please see "goapptrace --help" for more infomation about available commands.

## List of Subcommands
TODO: update command lists
```text
build - Compile packages using the "go build" command
run   - Compile and run the Go application using the "go run" command

log ls        - Print log IDs
log cat [id]  - Print logs

----- Advanced commands -----

target ls                         - Show tracing targets
target add [name] [dirs/files]    - Add tracing targets. This targets will be added tracing code
target remove [name] [dirs/files] - Remove from tracing targets
target set-build [name] [cmds...] - Set the custom build processes instead of 'go build'
target set-run [name] [cmd...]    - Set the custom command for start the application instead of './exe'

trace on [name]     - Insert tracing codes to targets
trace off [name]    - Remove tracing codes from targets
trace status [name] - Show status of tracer
trace start [name]  - Start tracing of running processes. It must be added tracing codes before processes started
trace stop [name]   - Stop tracing of running processes

proc build [name] - Start a command that defined by "goapptrace target set-build"
proc run [name]   - Start a command that defined by "goapptrace target set-run"

```

## TODO
* Reduce traffic between tracee and server.
* Reduce memory usage.
* Collect more information from runtime.ReadTrace().
* Add "targets" API to API spec.
* Implement TUI completely.
* Implement CLI completely.
* Create tests for tracer, builder and CLI commands.
* Create documents.
* Release a stable version.
* Improve performance.
* Enable linters that are currently disabled.

## Copyright and license
These codes are written by yuuki \<https://github.com/yuuki0xff\>.
Codes released under the MIT license.  
