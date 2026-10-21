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

### Benchmarking against Postgres

`BenchmarkSelectorPostgres` (`benchmark_postgres_test.go`) runs the selectors against a real
Postgres, built the way a node builds them: a generated `core.yaml` read by the FSC config
provider, the multiplexed Postgres driver, the `tokendb`/`tokenlockdb`/`ttxdb` store service
managers, and `sherdlock.NewService` / `simple.NewService` over a `token.ManagementService`.

```bash
make testing-docker-images                      # once: provides the postgres image
make benchmark-selector-postgres                # everything, 100 iterations per case
make benchmark-selector-postgres BENCH_FILTER='hot-wallet/' BENCH_TIME=500x
go test ./token/services/selector/ -run TestBenchPostgres -v   # check the harness itself
```

Without Docker the benchmark and the harness tests skip; with `PANURUS_REQUIRE_POSTGRES` set they
fail instead. They also skip under `-short`.

Environment: `PANURUS_SELECTOR_BENCH_MATRIX=full` runs the full sherdlock cross product
(fetcher x lock strategy x batch locking) in place of the one-axis sweep;
`PANURUS_SELECTOR_BENCH_LARGE=1` adds a 1M-token wallet to `wallet-size`;
`PANURUS_SELECTOR_BENCH_RETRY=5s` sets `token.selector.retryInterval` (the benchmarks default to 20ms
so contended cases do not spend their run asleep; 5s is the production default).

Each case reports `selects/sec`, `p50-ms`/`p95-ms`/`p99-ms` of `Select` alone, `contended/op`
(requests that ended as locked or insufficient funds: expected under contention, not an error),
`lockRT/op` (Postgres lock round trips) and `uniqueViol/op`, followed by the sherdlock metrics.

See [docs/services/selector.md](../../../docs/services/selector.md#benchmarking-against-postgres)
for the configurations and workloads.
