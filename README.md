# enigmap-go

`enigmap-go` is an oblivious key-value map implemented in Go. It stores an
AVL tree in Path ORAM and groups consecutive ORAM tree levels into physical
Large Buckets.

## Run the example

The command-line example inserts 1,000 entries, overwrites them, and verifies
the resulting values. The currently supported configurations are:

```sh
# Direct AVL access backed by memory
go run ./cmd/enigmap \
  -access direct -storage memory -encryption none

# Path ORAM backed by encrypted memory
go run ./cmd/enigmap \
  -access oram -storage memory -encryption aes-gcm

# Path ORAM backed by an encrypted file
go run ./cmd/enigmap \
  -access oram -storage file -encryption aes-gcm \
  -file enigmap-oram.db
```

## Test and benchmark

Run the regular test suite with:

```sh
go test ./...
```

The `latency` build tag enables the in-memory and file-backed ORAM latency
benchmarks. The following command builds and runs the in-memory benchmark in an
EGo enclave:

```sh
name="bench-$(date +%Y%m%d-%H%M%S).log"
ego-go test -c -o bench.test . -tags=latency,ego_largeheap
ego sign bench.test
ego run ./bench.test \
  -test.run='^$' \
  -test.benchmem \
  -test.benchtime=10x \
  -test.timeout=0 \
  -test.bench='^BenchmarkORAMMemoryStoreLatency$' 2>&1 | tee "$name"
```

This benchmark reports initialization, insertion, overwrite, and lookup
latency for ORAM capacities of `2^20-2`, `2^23-2`, and `2^26-2` nodes. The
in-memory benchmark excludes AES-GCM and filesystem I/O.

## Parameters

The current compile-time and default parameters are:

| Parameter | Value | Definition |
| --- | ---: | --- |
| Key size | 32 bytes | `config/fixed_size.go` |
| Value size | 1 byte | `config/fixed_size.go` |
| Bucket capacity (`Z`) | 5 blocks | `nodestore/eoram/bucket.go` |
| Stash capacity | 105 blocks | `nodestore/eoram/client.go` |
| Security target for the stash bound | 128 bits | `nodestore/eoram/client.go` |
| Levels per Large Bucket | 3 | `nodestore/eoram/bucket.go` |
| Buckets per Large Bucket | 7 | derived as `2^3-1` |
| Default Large Bucket cache | 50 entries | `nodestore/eoram/bucket.go` |

`maxNodes` determines the ORAM tree size. The implementation rounds
`maxNodes + 2` up to a power-of-two leaf count and pads the number of tree
levels to a multiple of `LevelsPerLargeBucket`.
