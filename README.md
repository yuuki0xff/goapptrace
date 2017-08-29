# goapptrace - function call tracer for golang

## Installation
```bash
$ go get github.com/yuuki0xff/goaptrace
```

## Usage
```bash
$ goapptrace proc run /path/to/project/dir
$ goapptrace log show
```

## List of Subcommands
```text
target ls                         - show tracing targets
target add [name] [dirs/files]    - add tracing targets. this targets will be added tracing code
target remove [name] [dirs/files] - remove from tracing targets
target set-build [name] [cmds...] - set the custom build processes instead of 'go build'

trace on [name]     - insert tracing codes to targets
trace off [name]    - remove tracing codes from targets
trace status [name] - show status of tracer
trace start [name]  - start tracing of running processes. it must be added tracing codes before processes started
trace stop [name]   - stop tracing of running processes

proc build [name] - build with tracing codes
proc run [name]   - start processes, and start tracing

log ls [name]   - show log names and histories
log show [name] - show logs on web browser
```

## Copyright and license
These codes are written by yuuki \<https://github.com/yuuki0xff\>.
Codes released under the MIT license.  

License of packages bundled with binaries are listed below:

* [Bulma](http://bulma.io/) : MIT
* [Font Awesome](http://fontawesome.io/) : SIL OFL 1.1 (font) and MIT License (CSS)
* [jQuery](https://jquery.org/) : MIT
* [TypeScript](https://www.typescriptlang.org/) : Apache 2.0
* [svg.js](https://svgdotjs.github.io/) : MIT
* [ng-content-editable](https://github.com/Vizir/ng-contenteditable) : MIT
