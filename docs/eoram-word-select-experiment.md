# EORAM fixed-word select experiment

## Result

The fivefold Go/C++ gap was caused mainly by the cost of one Go block select,
not by extra AVL or ORAM loop iterations. Specializing the configured 8-byte
key/value block into eleven 64-bit mask operations and updating the destination
in place reduced strong-oblivious Put latency below the public C++ result on the
same host.

| Entries | Original Go Z=5 | Go Z=4 before word select | Final Go | Public C++ |
|---:|---:|---:|---:|---:|
| 1,000 | 6.03 ms | 3.43 ms | 0.654-0.657 ms | 0.792 ms |
| 3,000 | 10.09 ms | 5.57 ms | 0.985-0.989 ms | 1.135 ms |
| 10,000 | - | 1.784 ms (first word version) | 1.445 ms | 1.574 ms |

The final operation breakdown with cache capacity 8192 was:

| Entries | New Put | Overwrite | Get |
|---:|---:|---:|---:|
| 1,000 | 0.657 ms | 0.647 ms | 0.317 ms |
| 3,000 | 0.986 ms | 0.970 ms | 0.483 ms |

Measurements are non-SGX, file-backed tmpfs runs and exclude final
`Close`/`Flush`. The C++ comparison also uses an in-memory backend, disables
Large Bucket encryption, and stops timing before teardown. These numbers must
not be presented as reproducing the paper's SGX experiment.

## Selected parameters

- `BucketSize (Z) = 4`, matching the paper and public implementation.
- `LevelsPerLargeBucket = 4` (15 logical buckets per physical record).
- key = 8 bytes, value = 8 bytes, both centralized in `config`.
- AES-GCM remains enabled for normal operation.
- cache capacity = 8192 for the C++ comparison. The production default remains
  50 and is intentionally reported separately.
- cryptographically secure random leaf selection remains enabled.

At this scale cache capacity 8192 holds the complete physical working set:
273 misses at 1,000 entries and 4,369 misses at 3,000 and 10,000 entries, with
no timed dirty eviction. Capacity 50 does not: earlier optimized measurements
were about 1.43 ms at 1,000 and 3.04 ms at 3,000 because hit rates fell to
93.1% and 86.9%. Cache settings therefore must accompany every latency result.

## Changes and causal experiments

### Fixed representation and conditional move

`FixedValue` represents the configured eight-byte value as one word-selectable
array. `avl.Key` stores the big-endian byte sequence as one `uint64`, preserving
byte-lexicographic order while reducing selection to one XOR-mask expression.
The key comparison uses two `bits.Sub64` borrow results and is branch-free in
the inspected assembly.

The concrete fixed ORAM block is pointer-free and occupies eleven 64-bit words.
The specialized eviction scans exactly the same depth, stash entry, bucket slot,
and word counts for every value. Each destination word is updated as:

```text
current ^= (current ^ candidate) & (0 - condition)
```

Updating the path slot in place was important. Selecting into a temporary
88-byte block and then assigning it measured about 1.02 ms/Put at 1,000 and
1.61 ms at 3,000. In-place words measured 0.657 ms and 0.986 ms respectively.
The removed work was compiler-generated temporary zeroing/copying, not an ORAM
access or safety check.

Keeping the word selector as a helper also regressed the 1,000-entry result
from about 0.78 ms to 1.03 ms even though Go reported the call as inlined.
This shows that register allocation and hot-loop code shape matter beyond the
source-level loop count. The hot loop therefore contains the small word loop
directly; a separate helper remains only for layout and correctness tests.

### Secondary safe improvements

- `PrepareAccess` now selects, relocates, and marks the target sticky in one
  complete stash scan instead of two.
- Cached Large Buckets are updated in place. Caller buckets are still cloned at
  the ownership boundary, but the entire 15-bucket physical value is no longer
  cloned for every path write. This made pack=4 slightly faster than pack=3.
- Secure random leaf generation was measured at 12.2 ms for 36,003 positions
  at 1,000 entries and 124 ms for 480,003 positions at 10,000 entries. It is
  below 2% and is not the main bottleneck.
- With a small cache, disabling AES-GCM improved the optimized 1,000-entry run
  by roughly 14%. With a full cache it affects initial misses and final flush,
  not the steady-state timed operations. Encryption remains enabled.

### Rejected buffer reuse

Reusing path output slices initially broke the generic memory-storage test:
the storage retained the supplied slice and the next eviction cleared stored
data. Restricting reuse to batch storage fixed ownership correctness but slowed
1,000 entries from about 0.78 ms to 1.02 ms and 10,000 from 1.77 ms to about
2.42 ms. The retained buffers changed escape/allocation and generated-code
behavior enough to lose the intended benefit. This experiment was removed.

## C++ and paper differences

The paper's strong eviction uses three oblivious sorts, and the public C++
`ObliviousCPUTrace=true` path calls that implementation. Go retains its existing
greedy `L * Z * S` fixed-trace scan. The final primitive is fast enough for the
tested sizes, but the algorithms have different asymptotic behavior; the C++
sort design should be evaluated at larger capacities before claiming equivalent
scaling.

The public C++ random-position helper deterministically increments a counter
and contains a TODO for secure randomness. Reproducing that behavior would
seriously weaken ORAM security, so the Go comparison keeps `crypto/rand`.

## Security assessment

The committed optimized path does not add data-dependent branches, early exits,
or memory indices to strong eviction. It preserves AES-GCM and secure leaf
randomization. Raw `unsafe` word access is guarded by tests that require the
concrete block to be pointer-free and word-aligned. It is a portability and
maintenance risk, not an intended reduction in access-pattern protection.

These measurements describe the original 8-byte/8-byte dynamic-stash stage of
the branch. A later experiment changes the configuration to a 32-byte key,
one-byte value, and a fixed 179-slot stash; see
`docs/eoram-fixed-stash-32-1-experiment.md`. Go's compiler/runtime/GC are still
not designed to guarantee strict instruction- or cache-trace obliviousness.

The experiment-only non-oblivious eviction is explicitly unsafe. Its branches,
early bucket termination, and direct assignments reduced Put to 0.219 ms at
1,000 and 0.268 ms at 3,000, but reveal data-dependent CPU/cache traces. It must
not be enabled in production and is retained only as causal evidence.

## Reproduction

```sh
GOCACHE=/tmp/enigmap-go-build-cache go test ./...
GOCACHE=/tmp/enigmap-go-build-cache go test -tags latency \
  -run '^TestORAMOptimizedOperationExperiment$' -v -count=1
GOCACHE=/tmp/enigmap-go-build-cache go test -tags latency \
  -run '^TestORAMHotLoopExperiment$/^entries-10000$/^cache-8192$' -v -count=1
GOCACHE=/tmp/enigmap-go-build-cache go test -tags latency \
  -run '^TestORAMEvictionModeExperiment$' -v -count=1
```
