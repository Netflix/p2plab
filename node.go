package p2plab

import "context"

type NodeAPI interface {
	Get(ctx context.Context, id string) (Node, error)
}

type Node interface {
	SSH(ctx context.Context, opts ...SSHOpt) error
}

type NodeSet interface {
	Add(node Node)

	Remove(node Node)

	Slice() []Node

	Label(ctx context.Context, addLabels, removeLabels []string) error
}

type SSHOpt func(SSHSettings) error

type SSHSettings struct {
}
