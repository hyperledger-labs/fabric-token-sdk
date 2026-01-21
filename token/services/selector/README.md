# Token Selector

This package contains the token selector implementations.
The token selector is responsible to select tokens of a given type and for a given amount from a wallet.

## Available Implementations

- Simple
- Sherdlock (Default)

## Benchmarking
```go
go test -v -run=XXX -bench=BenchmarkSelectorSingle . -benchmem
go test -v -run=XXX -bench=BenchmarkSelectorParallel . -cpu=4,8,16,32,64,128,256,512 -benchmem

go test -ginkgo.v -ginkgo.trace .
```
