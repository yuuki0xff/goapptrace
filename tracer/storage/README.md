# Directory Layout
```
info.json
./meta/<name>.meta.json
./data/<name>.<number>.rawfunc.log
./data/<name>.<number>.func.log
./data/<name>.<number>.goroutine.log
./data/<name>.symbol
./data/<name>.index
```

* `<name>`: 16バイトの乱数 (hex表記)
* `<number>`: 0から始まる連番
