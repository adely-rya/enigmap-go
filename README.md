# enigmap-go

Go で実装した ENIGMAP（oblivious key-value map）です。AVL 木を Path ORAM に格納し、
複数の論理 ORAM bucket を 1 個の Large Bucket にまとめて、暗号化したファイルへ保存します。

## 実験パラメーターの変更場所

SGX/EGo の性能実験で変更する主な値は次の場所にあります。`const` を変更する項目は
コンパイル時に決まるため、変更後にバイナリまたはテストを再ビルドしてください。

| パラメーター | 現在値 | 変更場所 | 備考 |
| --- | ---: | --- | --- |
| 1 bucket 内の block 数 `Z` | 4 | `nodestore/eoram/bucket.go` の `BucketSize` | 論理 bucket の容量。直列化サイズ、stash 処理量、ORAM の性能に影響する |
| 1 Large Bucket にまとめる tree level 数 | 4 | `nodestore/eoram/bucket.go` の `LevelsPerLargeBucket` | Large Bucket 内の論理 bucket 数は `2^LevelsPerLargeBucket - 1`。現在は 15 個 |
| 既定の Large Bucket cache 数 | 50 | `nodestore/eoram/bucket.go` の `DefaultLargeBucketCacheCapacity` | 単位は「物理 Large Bucket の個数」。`NewORAM` が使用する。0 で cache 無効 |
| 実験ごとの Large Bucket cache 数 | テストごと | `NewORAMWithCacheCapacity` の第4引数、または各実験テストの配列 | コード全体の既定値を変えずに比較するときはこちらを変更する |
| key サイズ | 32 byte | `config/fixed_size.go` の `KeySize` | compile-time 固定。公開 API に渡す key は常にこの長さが必要 |
| value サイズ | 1 byte | `config/fixed_size.go` の `ValueSize` | compile-time 固定。`FixedValue` と block/Large Bucket の直列化サイズに影響する |
| ORAM の最大 node 数・実験件数 | テストごと | `NewORAM*` の `maxNodes` 引数、各 `*_test.go` の `entries` / `EntryCount` | `maxNodes` から leaf 数と論理/物理 bucket 数が自動計算される |
| stash 数 | 自動計算 | `profile.go` の `NewORAMForProfile` 第5引数 | stash 比較専用。通常は `NewORAM` / `NewORAMWithCacheCapacity` の自動値を使う |
| 暗号化の有無 | AES-GCM 有効 | `NewORAM*` または `profile.go` の `NewORAMForProfileUnencrypted` | 暗号化なしはコスト分離用であり、本番・安全性評価には使用しない |
| eviction 実装 | oblivious | `profile.go` の `EnableNonObliviousEvictionForProfile` | non-oblivious は比較実験専用で、CPU/cache trace を秘匿しない |

### Large Bucket の bucket 数を変える

`nodestore/eoram/bucket.go` の次の定数を変更します。

```go
LevelsPerLargeBucket = 4
BucketsPerLargeBucket = (1 << LevelsPerLargeBucket) - 1
```

通常は `LevelsPerLargeBucket` だけを変更します。`BucketsPerLargeBucket` と
`serializedLargeBucketSize` はそこから自動的に導出されます。例えば level 数が 3、4、5 の
場合、1 Large Bucket に含まれる論理 bucket はそれぞれ 7、15、31 個です。

この値を変えると、物理ファイルの record サイズ、暗号化単位、padding、物理 Large Bucket 数が
同時に変わります。過去の DB ファイルとはレイアウト互換にならないため、実験ごとに新しい
ファイルを作成してください。

### cache 数を変える

通常実行の既定値を変える場合は `DefaultLargeBucketCacheCapacity` を変更します。1 回の実験だけ
変える場合は、次のように `NewORAMWithCacheCapacity` へ明示します。

```go
store, err := enigmap.NewORAMWithCacheCapacity[eoram.FixedValue](
    filePath,
    maxNodes,
    key,
    8192, // plaintext Large Bucket cache の個数
)
```

cache は enclave 内に平文の Large Bucket を保持します。SGX 実験では cache 数を増やすほど
OCALL/file I/O は減りますが、EPC/heap 使用量は増えます。容量は byte 数ではないため、概算
メモリ量は `cacheCapacity * serializedLargeBucketSize` に cache 管理領域を加えた値です。

ORAM tree の物理 Large Bucket 数は `maxNodes` から決まり、概略は次の順で計算されます。

```text
requiredLeaves = maxNodes + 2
leafCount      = requiredLeaves 以上の最小の 2 の累乗
logicalBuckets = 2 * leafCount - 1
paddedLevels   = tree level 数を LevelsPerLargeBucket の倍数へ切り上げ
physicalLargeBuckets = (2^paddedLevels - 1) / BucketsPerLargeBucket
```

working set 全体を cache する比較では、実行ログの cache miss 数や上式で得た
`physicalLargeBuckets` を目安にしてください。

## 実験ケースの変更場所

- `oram_hotloop_experiment_test.go`: 件数、cache 数、暗号化有無、eviction 方式、操作別計測
- `oram_cache_profile_test.go`: cache 容量と stash 容量の比較
- `oram_latency_test.go`: file-backed ORAM の Put/Get レイテンシ。件数は
  `oramLatencyEntryCount`
- `latency_test.go`: ORAM を使わない memory AVL の基準計測。`entryCounts`、round 数、最低操作数
- `config/fixed_size.go`: SGX 用の固定 key/value サイズ

実験用テストには `latency` build tag が必要です。個別ケースは `-run` で絞り込みます。

```sh
GOCACHE=/tmp/enigmap-go-build-cache go test ./...
GOCACHE=/tmp/enigmap-go-build-cache go test -tags latency \
  -run '^TestORAMHotLoopExperiment$/^entries-10000$/^cache-8192$' \
  -v -count=1
```

`NewORAMForProfileUnencrypted` と `EnableNonObliviousEvictionForProfile` は原因切り分け専用です。
SGX の安全な構成の測定値と混同しないよう、結果には key/value サイズ、`BucketSize`、
`LevelsPerLargeBucket`、cache 数、stash 数、暗号化・eviction 設定を併記してください。
