package avl

type NodeID uint64

const NullAddress NodeID = 0

type Balance uint8

const (
	BalanceLeft     Balance = iota //左に１つ子が多い
	BalanceRight                   //右に1つ子が多い
	BalanceBalanced                //左右の高さが同じ
)

func selectNodeID(current, candidate NodeID, condition int) NodeID { //conditionが0ならcurrent 1ならcandinateが選択
	mask := uint64(0) - uint64(condition&1)
	return NodeID(
		(uint64(current) &^ mask) |
			(uint64(candidate) & mask),
	)
}

func nodeIDPresent(id NodeID) int {
	value := uint64(id)
	return int((value | (0 - value)) >> 63)
}

type Node[V any] struct {
	Key   Key // key component of ptr in the paper
	Value V   // v in the paper

	Balance Balance // b in the paper

	Left  NodeID // address component of lptr in the paper
	Right NodeID // address component of rptr in the paper
	Valid uint8
}
