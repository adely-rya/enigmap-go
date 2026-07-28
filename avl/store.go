package avl

type NodeStore[V any] interface {
	Allocate(node Node[V]) (NodeID, error)    // 新たなノードを追加
	Load(id NodeID) (Node[V], error)          // ノードの情報を取得
	Update(id NodeID, node Node[V]) error     // ノードの情報を変更
	FreeMaybe(id NodeID, condition int) error // conditionが1なら候補ノードを解放
}
