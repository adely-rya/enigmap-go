# Fixed-stash 32-byte key / 1-byte value experiment

## Conditions

- branch: `feature/profile-eoram-word-select`
- key: 32 bytes, stored as four big-endian `uint64` words
- value: 1 byte
- stash: fixed capacity 179; all slots, including invalid slots, are scanned
- `Z=4`, `LevelsPerLargeBucket=4`
- AES-GCM and `crypto/rand` remain enabled
- sequential new Put; final Close/Flush excluded from the timed interval

The stash capacity is derived from the existing 128-bit security calculation,
not from the current number of valid entries. Search, update, pin/unpin, path
insertion, and strong eviction iterate over all 179 slots. Eviction does not
compact the slice; invalid slots are reused through fixed scans.

## Fixed-word implementation

The 32-byte key is selected as four words and compared lexicographically by
scanning all four words. The one-byte value uses a byte mask. The complete
pointer-free concrete ORAM block remains selected as raw `uint64` words in the
strong eviction loop.

An initial fixed-stash implementation left path insertion on the nested generic
`Block.Select` path. At 10,000 entries this took 207.7 seconds of the 254.1
second Put interval. Specializing path insertion to raw word selection, while
retaining its two full 179-slot scans per path block, reduced total time to
83.2 seconds. This is the same causal result as strong eviction: fixed loop
counts alone are insufficient when each iteration recursively selects every
field through generic methods.

## Results

| Entries | Cache | Put total | Put/op | Max valid stash |
|---:|---:|---:|---:|---:|
| 1,000 | 8,192 | 4.553 s | 4.553 ms | 15 |
| 10,000 | 8,192 | 83.186 s | 8.319 ms | 16 |
| 100,000 | 69,905 | 1,167.958 s | 11.680 ms | 20 |

The 100,000-entry tree has 69,905 physical Large Buckets after four-level
padding. Cache capacity 69,905 was used to retain the physical working set,
matching the earlier method of isolating ORAM CPU cost. It consumed far more
memory than the normal default capacity 50 and is a comparison setting, not a
recommended production default.

The 100,000-entry phase breakdown was:

| Phase | Time |
|---|---:|
| path insertion into fixed stash | 527.475 s |
| strong eviction | 547.990 s |
| prepare/search | 22.881 s |
| update | 23.632 s |
| path read | 21.948 s |
| path write | 13.401 s |
| secure random positions | 1.802 s |

There were 5,200,001 ORAM accesses, 930,800,179 scans each for prepare,
update, and unpin, and 16,754,403,222 eviction stash iterations. Eviction made
67,017,612,888 block-slot selections. The cache recorded 69,905 initial misses,
119,530,118 hits, and no timed backing write or dirty eviction.

## Cache=8192 observation

The same 100,000-entry test with cache 8192 exceeded Go's default ten-minute
test timeout. At timeout it was performing AES-GCM serialization during dirty
LRU eviction. That incomplete run is not reported as latency. It demonstrates
that the public implementation's default cache size no longer holds the
working set at this capacity; I/O and encryption become additional costs.

## Interpretation and security

The maximum number of valid blocks observed was only 20, but every operation
still scanned 179 slots. This removes dependence of stash loop length on its
current occupancy and costs roughly an order of magnitude versus the earlier
dynamic-stash 8-byte/8-byte experiment. Comparisons are not payload-only because
both the stash policy and payload sizes changed.

No non-oblivious early exit, data-dependent stash index, deterministic leaf
position, or encryption bypass is enabled. The optimized path performs the
same fixed scans with cheaper word operations. However, this is still not a
formal CPU/cache-trace guarantee: Go compilation, GC, maps and the cache layer
are outside such a proof, and error/overflow handling may terminate publicly.

## Reproduction

```sh
GOCACHE=/tmp/enigmap-go-build-cache go test ./...
GOCACHE=/tmp/enigmap-go-build-cache go test -timeout 40m -tags latency \
  -run '^TestORAMHotLoopExperiment$/^entries-100000$/^cache-69905$' \
  -v -count=1
```
