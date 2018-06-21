# Benchmarks of Go serialization methods

[![Gitter chat](https://badges.gitter.im/alecthomas.png)](https://gitter.im/alecthomas/Lobby)

This is a test suite for benchmarking various Go serialization methods.

## Tested serialization methods

- [encoding/gob](http://golang.org/pkg/encoding/gob/)
- [encoding/json](http://golang.org/pkg/encoding/json/)
- [github.com/alecthomas/binary](https://github.com/alecthomas/binary)
- [github.com/davecgh/go-xdr/xdr](https://github.com/davecgh/go-xdr)
- [github.com/Sereal/Sereal/Go/sereal](https://github.com/Sereal/Sereal)
- [github.com/ugorji/go/codec](https://github.com/ugorji/go/tree/master/codec)
- [gopkg.in/vmihailenco/msgpack.v2](https://github.com/vmihailenco/msgpack)
- [labix.org/v2/mgo/bson](https://labix.org/v2/mgo/bson)
- [github.com/tinylib/msgp](https://github.com/tinylib/msgp) *(code generator for msgpack)*
- [github.com/golang/protobuf](https://github.com/golang/protobuf) (generated code)
- [github.com/gogo/protobuf](https://github.com/gogo/protobuf) (generated code, optimized version of `goprotobuf`)
- [github.com/DeDiS/protobuf](https://github.com/DeDiS/protobuf) (reflection based)
- [github.com/google/flatbuffers](https://github.com/google/flatbuffers)
- [github.com/hprose/hprose-go/io](https://github.com/hprose/hprose-go)
- [github.com/glycerine/go-capnproto](https://github.com/glycerine/go-capnproto)
- [zombiezen.com/go/capnproto2](https://godoc.org/zombiezen.com/go/capnproto2)
- [github.com/andyleap/gencode](https://github.com/andyleap/gencode)
- [github.com/linkedin/goavro](https://github.com/linkedin/goavro)

## Running the benchmarks

```bash
go get -u -t
go test -bench='.*' ./
```

Shameless plug: I use [pawk](https://github.com/alecthomas/pawk) to format the table:

```bash
go test -bench='.*' ./ | pawk -F'\t' '"%-40s %10s %10s %s %s" % f'
```

## Recommendation

If performance, correctness and interoperability are the most
important factors, [gogoprotobuf](https://gogo.github.io/) is
currently the best choice. It does require a pre-processing step (eg.
via Go 1.4's "go generate" command).

But as always, make your own choice based on your requirements.

## Data

The data being serialized is the following structure with randomly generated values:

```go
type A struct {
	ID        int64
	Tag       uint8
	Timestamp int64
	Frames    []uint64
	GID       int64
	TxID      uint64
}
```


## Results

2018-06-20 Results with Go 1.10.3 on a Intel Core i7-7700 CPU @ 3.60GHz:

```
goos: linux
goarch: amd64
pkg: bitbucket.org/yuuki0xff/goapptrace-codec-benchmarks
BenchmarkMsgpMarshal-8                          10000000               226 ns/op             160 B/op          1 allocs/op
BenchmarkMsgpUnmarshal-8                         5000000               296 ns/op             128 B/op          2 allocs/op
BenchmarkVmihailencoMsgpackMarshal-8              500000              2204 ns/op             336 B/op          5 allocs/op
BenchmarkVmihailencoMsgpackUnmarshal-8            500000              2808 ns/op             512 B/op         15 allocs/op
BenchmarkJsonMarshal-8                           1000000              2450 ns/op            1160 B/op          9 allocs/op
BenchmarkJsonUnmarshal-8                          500000              2275 ns/op             464 B/op          7 allocs/op
BenchmarkEasyJsonMarshal-8                       2000000               919 ns/op             720 B/op          4 allocs/op
BenchmarkEasyJsonUnmarshal-8                     2000000               880 ns/op             128 B/op          2 allocs/op
BenchmarkBsonMarshal-8                           1000000              1972 ns/op             712 B/op         20 allocs/op
BenchmarkBsonUnmarshal-8                          500000              2803 ns/op             447 B/op         43 allocs/op
BenchmarkGobMarshal-8                            2000000               724 ns/op              32 B/op          1 allocs/op
BenchmarkGobUnmarshal-8                          2000000               838 ns/op             192 B/op          4 allocs/op
BenchmarkXdrMarshal-8                            1000000              1370 ns/op             408 B/op         26 allocs/op
BenchmarkXdrUnmarshal-8                          2000000               965 ns/op             208 B/op          9 allocs/op
BenchmarkSerealMarshal-8                          500000              2819 ns/op            1072 B/op         33 allocs/op
BenchmarkSerealUnmarshal-8                        500000              2182 ns/op             864 B/op         26 allocs/op
BenchmarkBinaryMarshal-8                         1000000              1779 ns/op             488 B/op         28 allocs/op
BenchmarkBinaryUnmarshal-8                       1000000              1845 ns/op             384 B/op         25 allocs/op
BenchmarkCapNProtoMarshal-8                      3000000               439 ns/op              56 B/op          2 allocs/op
BenchmarkCapNProtoUnmarshal-8                    3000000               552 ns/op             272 B/op          8 allocs/op
BenchmarkCapNProto2Marshal-8                     2000000               650 ns/op             276 B/op          3 allocs/op
BenchmarkCapNProto2Unmarshal-8                   2000000               923 ns/op             392 B/op          8 allocs/op
BenchmarkHproseMarshal-8                        10000000               207 ns/op             117 B/op          0 allocs/op
BenchmarkHproseUnmarshal-8                       5000000               280 ns/op              96 B/op          2 allocs/op
BenchmarkProtobufMarshal-8                       2000000               886 ns/op             280 B/op         10 allocs/op
BenchmarkProtobufUnmarshal-8                    10000000               211 ns/op              64 B/op          1 allocs/op
BenchmarkGoprotobufMarshal-8                     2000000               572 ns/op             328 B/op          5 allocs/op
BenchmarkGoprotobufUnmarshal-8                   2000000               842 ns/op             488 B/op          8 allocs/op
BenchmarkGogoprotobufMarshal-8                  10000000               180 ns/op              63 B/op          1 allocs/op
BenchmarkGogoprotobufUnmarshal-8                 5000000               324 ns/op             184 B/op          5 allocs/op
BenchmarkGencodeMarshal-8                       10000000               169 ns/op             112 B/op          1 allocs/op
BenchmarkGencodeUnmarshal-8                     10000000               171 ns/op             128 B/op          2 allocs/op
BenchmarkGencodeUnsafeMarshal-8                 10000000               115 ns/op             112 B/op          1 allocs/op
BenchmarkGencodeUnsafeUnmarshal-8               10000000               142 ns/op             128 B/op          2 allocs/op
PASS
ok      bitbucket.org/yuuki0xff/goapptrace-codec-benchmarks     67.344s
```

## Issues


The benchmarks can also be run with validation enabled.

```bash
VALIDATE=1 go test -bench='.*' ./
```

Unfortunately, several of the serializers exhibit issues:

1. **(minor)** BSON drops sub-microsecond precision from `time.Time`.
3. **(minor)** Ugorji Binc Codec drops the timezone name (eg. "EST" -> "-0500") from `time.Time`.

```
--- FAIL: BenchmarkBsonUnmarshal-8
    serialization_benchmarks_test.go:115: unmarshaled object differed:
        &{20b999e3621bd773 2016-01-19 14:05:02.469416459 -0800 PST f017c8e9de 4 true 0.20887343719329818}
        &{20b999e3621bd773 2016-01-19 14:05:02.469 -0800 PST f017c8e9de 4 true 0.20887343719329818}
--- FAIL: BenchmarkUgorjiCodecBincUnmarshal-8
    serialization_benchmarks_test.go:115: unmarshaled object differed:
        &{20a1757ced6b488e 2016-01-19 14:05:15.69474534 -0800 PST 71f3bf4233 0 false 0.8712180830484527}
        &{20a1757ced6b488e 2016-01-19 14:05:15.69474534 -0800 -0800 71f3bf4233 0 false 0.8712180830484527}
```

All other fields are correct however.

Additionally, while not a correctness issue, FlatBuffers, ProtoBuffers and Cap'N'Proto do not
support time types directly. In the benchmarks an int64 value is used to hold a UnixNano timestamp.
