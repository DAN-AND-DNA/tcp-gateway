# ZGame Message Encoder And Decoder

## Benchmark

```batch
go test -bench . go-dmm-connect/protocol
```

```plain
goos: windows
goarch: amd64
pkg: go-dmm-connect/protocol
BenchmarkUnmarshal-4              666724              1626 ns/op
BenchmarkJsonUnmarshal-4           75957             15736 ns/op
BenchmarkGobUnmarshal-4            37854             32335 ns/op
BenchmarkMarshal-4                571501              2063 ns/op
BenchmarkJsonMarshal-4            260862              4489 ns/op
BenchmarkGobMarshal-4             155853              7185 ns/op
PASS
ok      go-dmm-connect/protocol 7.814s

goos: windows
goarch: amd64
pkg: go-dmm-connect/encoding
cpu: Intel(R) Core(TM) i5-6500 CPU @ 3.20GHz
BenchmarkUnmarshal-4              662664              1626 ns/op
BenchmarkJsonUnmarshal-4           90348             13211 ns/op
BenchmarkGobUnmarshal-4            42669             28235 ns/op
BenchmarkMarshal-4                562621              2315 ns/op
BenchmarkJsonMarshal-4            316850              3939 ns/op
BenchmarkGobMarshal-4             192806              6216 ns/op
PASS
ok      go-dmm-connect/encoding 7.868s
```
# 呆羊测试结果
```shell
goos: windows
goarch: amd64
pkg: gateway/pkg/encoding
cpu: AMD Ryzen 5 5600G with Radeon Graphics
BenchmarkUnmarshal
BenchmarkUnmarshal-12            1457641               816.2 ns/op
BenchmarkJsonUnmarshal
BenchmarkJsonUnmarshal-12         136041              8613 ns/op
BenchmarkGobUnmarshal
BenchmarkGobUnmarshal-12           75327             15858 ns/op
BenchmarkMarshal
BenchmarkMarshal-12               944034              1219 ns/op
BenchmarkJsonMarshal
BenchmarkJsonMarshal-12           480974              2465 ns/op
BenchmarkGobMarshal
BenchmarkGobMarshal-12            352036              3440 ns/op
PASS
```
