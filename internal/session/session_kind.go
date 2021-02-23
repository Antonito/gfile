package session

type Kind uint8

const (
	/// KindMaster represents a master/main node.
	KindMaster Kind = iota
	/// KindNode represents a node.
	KindNode
)
