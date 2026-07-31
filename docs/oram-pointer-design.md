# ENIGMAP ORAM pointer設計

## 目的

`NodeStore`へPath ORAMを追加する際に、標準的な集中型position mapではなく、
ENIGMAP論文と公開C実装の分散position方式を採用する。

参照元:

- 論文: workspaceの`paper/2022-1083.pdf`
- C実装: workspaceの`EnigMap/ods/otree/`、`EnigMap/ods/oram/pathoram/`
- Go Path ORAM: workspaceの`oram/oram/pathoram/`

## 標準Path ORAMとの差分

既存Go Path ORAMは`PositionMap[logical ID]`を集中管理し、1回の`Access`内で
position map swap、path read、stash更新、path evictionまで完結する。

ENIGMAP方式では次の点が異なる。

- pointerはlogical addressとORAM positionの組を保持する。
- root positionは`Tree.root`、child positionは親node内の`Left`/`Right`が所有する。
- node accessは`BeginReadOp`と`FinishOramOp`に分かれる。
- BeginとFinishの間で、次にアクセスするchildへ新positionを割り当て、親nodeへ保存する。
- Finish時にstash上のnodeを書き換え、読み込んだpathへevictする。
- stash上の一部blockをcached/stickyにしてeviction対象外とし、Put終了時に解除する。
- stashのcached/stickyと外部ファイルのLarge Bucket cacheは別の仕組みである。

## Pointer

```go
type NodeID uint64
type ORAMPosition uint64

type Pointer struct {
    Address  NodeID
    Position ORAMPosition
}
```

pointerのnull判定とlogical nodeの同一性判定には`Address`を使う。positionはアクセスごとに
変わるため、AVLのpivot、parent、child、grandchildの同一性判定に含めない。
一方、constant-time pointer選択ではAddressとPositionの両方を選択する。

## Node accessの境界

最終的な`NodeStore`は、現在の`Load`/`Update`から次のBegin/Finish型へ移行する。

```go
type AccessToken struct {
    Address NodeID
}

type NodeStore[V any] interface {
    NewPosition() (ORAMPosition, error)
    BeginAccess(Pointer, ORAMPosition) (Node[V], AccessToken, error)
    FinishAccess(AccessToken, Node[V]) error
    Allocate(Node[V]) (Pointer, error)
    FreeMaybe(Pointer, condition int) error
    UncacheAll()
}
```

`BeginAccess`はold positionのpathをstashへ読み、対象blockをnew positionへ再配置する。
呼び出し側はnodeを調査・更新し、次に進むchild pointerへ次回のnew positionを保存する。
`FinishAccess`はnodeをstashへ戻し、path evictionを完了する。

メモリNodeStoreも同じAPIを実装し、positionとevictionだけをno-opにする。これにより、
Get/Putの制御フローと固定access scheduleをバックエンド間で共通化する。

## Putの二種類の分割

Putには異なる粒度の二つの分割がある。

1. AVL操作全体は、挿入・rotation分類の第1パスと、
   balance更新・rotationの第2パスに分かれる。
2. 両パスの各node accessは、Begin、node計算・書き換え、Finish・evictionに分かれる。

両パスは結果にかかわらず`maxDepth`回アクセスする。実nodeの走査終了後はfake nodeで
paddingし、overwriteの場合も第2パスを省略しない。

## 移行順序

1. `Pointer{Address, Position}` とconstant-time helperを追加する。
2. `Node.Left`/`Node.Right`を`Pointer`へ変更する。
3. `Tree.root`/`Tree.fake`を`Pointer`へ変更する。
4. Put metadata内のnode参照を`Pointer`へ変更する。
5. `NodeStore`をBegin/Finish型へ変更する。
6. memory NodeStoreを新APIへ移行する。
7. Get、第1 Putパス、第2 Putパスの順にposition更新を組み込む。
8. `nodestore/eoram`へstash cached状態とLarge Bucket cacheを接続する。

1〜7は`feature/oram-pointer-foundations`で実施する。memory NodeStoreではposition生成と
`UncacheAll`をno-opとし、Begin/Finishをmapのread/writeへ接続する。8は後続変更とする。

## memory NodeStoreレイテンシ

Begin/Finish分割前後に、次の同一条件で`TestMemoryStoreLatency`を3回実行した。

```text
GOMAXPROCS=1
go test -tags latency -run '^TestMemoryStoreLatency$' -count=3 -v
```

各3回の中央値は次のとおり。単位は1操作あたりのµs。

| entries | new Put（前→後） | overwrite（前→後） | Get（前→後） |
| ---: | ---: | ---: | ---: |
| 10 | 6.626 → 4.452 | 6.404 → 4.384 | 2.459 → 1.983 |
| 100 | 9.836 → 7.070 | 9.784 → 7.007 | 3.504 → 3.065 |
| 1,000 | 13.489 → 10.299 | 13.336 → 10.183 | 4.622 → 4.943 |
| 10,000 | 18.795 → 12.280 | 19.591 → 11.286 | 6.471 → 5.184 |

Begin/Finish分割による大幅な性能低下は観測されなかった。短縮値も出ているが、
memory mapの本質的な処理量を減らす変更ではないため、CPU状態などの測定揺れを含む。
ここでは高速化とは断定せず、性能回帰なしと判断する。
