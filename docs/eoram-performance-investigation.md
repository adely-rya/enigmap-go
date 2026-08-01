# EORAM performance investigation

## Scope

This report compares the Go EORAM implementation with the ENIGMAP paper and
the public C++ implementation. Measurements were run on the same host, but the
paper's SGX results are not directly comparable to these small non-SGX tests.
The paper evaluates much larger databases, uses a 3992-byte page, and reports
that locality layout plus page- and bucket-level caches improve total time by
2x to 5.1x.

## Relevant reference behavior

The paper and public C++ implementation use these important mechanisms:

- Path ORAM bucket capacity `Z=4`.
- Sticky stash entries avoid ORAM reads during AVL Insert's second pass.
- Strong eviction in the paper and public C++ code performs three oblivious
  sorts. Go's current strong path uses a fixed-trace greedy
  `path length * Z * stash length` scan instead.
- Physical caching has two levels: an encrypted page cache outside the enclave
  and an ORAM-bucket cache inside it.
- The public C++ defaults are four levels per Large Bucket, an 8192-entry
  Large Bucket cache, an 8192-entry backing-page cache, and Large Bucket
  encryption disabled.

The Go implementation before this experiment used `Z=5`, three levels per
Large Bucket, a much smaller decrypted Large Bucket cache, AES-GCM, and no
explicit encrypted page cache below it.

## Loop comparison

AVL traversal counts are already aligned:

| Entries | Go ORAM accesses per Insert | C++ max AVL depth | Expected two passes |
|---:|---:|---:|---:|
| 1,000 | 32 | 16 | 32 |
| 3,000 | 38 | 19 | 38 |

The remaining difference is primarily inside each ORAM access.

| Operation | Go | Public C++ | Effect |
|---|---|---|---|
| Read path into stash | `L * Z` appends | `L * Z` appends | Equivalent |
| Find and pin target | two stash scans | one stash scan | Go has one extra scan |
| Update and unpin | separate scans | combined CMOV scan | Go has an extra scan |
| Strong eviction | `L * S * Z` full block selects | three oblivious bitonic sorts | Different algorithms; Go's inner primitive was initially much more expensive |
| Conditional move | nested Go generic `Select` and bit masks | inlined field CMOV/assembly | Go is substantially more expensive per iteration |

Here `L` is path length and `S` is the stash length immediately before
eviction. The extra non-eviction stash scans account for only a few percent of
the measured time. Strong eviction accounts for about 90% to 94% once the
Large Bucket cache can hold the working set.

## Measurements

All Go timings below measure sequential new Put and exclude final Close/Flush.

### Original Go settings (`Z=5`, pack=3, AES-GCM)

| Entries | Cache | Put/op | Eviction total | Block selects | Cache hit rate |
|---:|---:|---:|---:|---:|---:|
| 1,000 | 30 | 6.03 ms | 4.84 s | 99.50 M | 86.5% |
| 1,000 | 8192 | 5.21 ms | 4.85 s | 99.50 M | 99.9% |
| 3,000 | 30 | 10.09 ms | 23.93 s | 492.96 M | 82.0% |
| 3,000 | 8192 | 8.44 ms | 23.75 s | 492.96 M | 99.8% |

Increasing the cache to 8192 improves time by about 14% to 16%, so cache
thrashing is real but is not the whole fivefold gap.

### Match the paper's `Z=4`

| Entries | Cache | Put/op | Max stash after path read | Block selects |
|---:|---:|---:|---:|---:|
| 1,000 | 8192 | 3.43 ms | 47 | 64.11 M |
| 3,000 | 8192 | 5.57 ms | 55 | 317.31 M |
| 1,000 | 30 | 4.13 ms | 47 | 64.11 M |
| 3,000 | 30 | 6.89 ms | 55 | 317.31 M |

Changing only `Z` from 5 to 4 reduces time by about 34%, not merely 20%.
The path contributes fewer blocks to the stash, so both `Z` and `S` shrink;
the total number of full block selects falls by about 36%.

### Public C++ implementation (`Z=4`, pack=4, cache=8192, no encryption)

| Entries | Insert/op | Get/op |
|---:|---:|---:|
| 1,000 | 0.792 ms | 0.380 ms |
| 3,000 | 1.135 ms | 0.552 ms |

The C++ comparison uses its in-memory backend and no Large Bucket encryption.
It nevertheless uses the default strong `ObliviousCPUTrace=true` Path ORAM
eviction. In that branch `FinishAccess` calls `EvictPath`, which executes three
oblivious sorts; the triple-loop `WriteBucket` path belongs to the
`ObliviousCPUTrace=false` external-oblivious mode. The Go/C++ Insert gap after
matching only `Z` remained about 4.3x at 1,000 and 4.9x at 3,000.

### Encryption isolation (`Z=4`, pack=3, cache=30, file backed)

| Mode | 1,000 Put/op |
|---|---:|
| AES-GCM | 3.83 ms |
| Plain file | 3.60 ms |

Removing AES-GCM improves this paired run by about 6%. Encryption is measurable
but not the main cause.

### Large Bucket packing isolation

Changing pack levels from 3 to the C++ default 4 slightly regressed Go:

- cache 8192: approximately 3.43 ms to 3.46 ms/op;
- cache 30: approximately 4.13 ms to 4.31 ms/op.

The C++ cache updates a Large Bucket through a reference. Go `WriteBatch`
clones the whole Large Bucket before changing its inner buckets. Increasing
the pack from 7 to 15 logical buckets therefore increases Go copying enough
to offset fewer physical cache lookups. Pack=3 remains preferable until the
cache update path is changed to avoid whole-Large-Bucket cloning.

### Non-oblivious eviction control experiment

The experiment branch includes an explicit unsafe mode equivalent to the
public C++ `ObliviousCPUTrace=false` branch. It stops after filling a bucket and
uses ordinary branches/direct assignments.

| Entries | Strong Go | Non-oblivious Go | Strong eviction | Non-oblivious eviction |
|---:|---:|---:|---:|---:|
| 1,000 | 3.14 ms/op | 0.342 ms/op | 2.85 s | 0.035 s |
| 3,000 | 5.07 ms/op | 0.393 ms/op | 13.91 s | 0.094 s |

This mode is faster than the C++ benchmark, confirming that the AVL schedule,
ORAM access count, file layer, and Go language overhead outside strong eviction
are not the fivefold bottleneck. The dominant difference is the cost of the
hundreds of millions of full-structure constant-time selects. This unsafe mode
must not be used in production because branches, early termination, and direct
assignments reveal data-dependent CPU/cache traces.

## Conclusions and next changes

1. Keep `BucketSize=4`; it matches both the paper and public implementation and
   produced the largest safe improvement.
2. Size the decrypted Large Bucket cache for the experiment explicitly. A
   capacity of 30 or 50 cannot be compared with the C++ default 8192 and causes
   heavy thrashing at 1,000 and 3,000 entries.
3. Re-test pack=4 after eliminating whole-Large-Bucket clones. This follow-up
   was completed on `feature/profile-eoram-word-select`; pack=4 is slightly
   faster and is now the selected setting.
4. Keep AES-GCM enabled for real measurements; disabling it explains only a
   small part of the current gap.
5. Optimize the strong conditional exchange primitive. The likely productive
   direction is a fixed concrete ORAM node representation using word-sized
   conditional moves, with compiler assembly inspection and microbenchmarks.
   A generic nested `Selectable` implementation is convenient but too costly
   in the innermost `L * Z * S` loop.
6. Combining target selection with pinning and update with unpinning can remove
   two stash scans, but measured phase times show this is secondary.
7. Do not adopt the non-oblivious experiment mode as an optimization. It exists
   only as causal evidence.

The completed fixed-word follow-up, current measurements, and security notes
are in `docs/eoram-word-select-experiment.md`.
