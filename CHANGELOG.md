# Changelog

## v0.3.0-beta (Under development)
* __Breaking change__: Redesigned the goapptrace command.
* __Breaking change__: Removed the tracer API.
* __New feature__: Added ability to change configuration of "function call tracer" without restarting.
* __New feature__: Added local variable logger. (planned)
* __Improvement__: Added a tutorial. (planned)
* __Improvement__: Updated documentation. (planned)
* __Improvement__: Implemented "/log/{log-id}/watch" API.
* __Bug fix__: Fixed build error when building with package path.
* __Bug fix__: Fixed bug that "/log/{log-id}" API returns wrong data format

## v0.2.0-alpha (2018-04-01)
* __New feature__: Added SQL query API and CLI.
* __Improvement__: Improved error handlings of CLI.
* __Improvement__: Added _goapptrace\_debug_ command for dump binary logs.
* __Improvement__: Added an example code of R language.
* __Bug fix__: Fixed a bug on log server that memory usage increases infinitely when logging.
* __Bug fix__: Fixed goroutine leak on search API when connection closed by client.

## v0.1.0-alpha (2018-03-19)
* First release
