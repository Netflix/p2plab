package p2plab

import "context"

type NodeAPI interface {
	Get(ctx context.Context, id string) (Node, error)

	List(ctx context.Context) ([]Node, error)
}

type Node interface {
	SSH(ctx context.Context) error
}

type NodeSet interface {
	Label(ctx context.Context, labels ...string) error

	Add(node Node)

	Remove(node Node)

	Slice() []Node
}
