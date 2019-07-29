package node

import (
	"context"

	"github.com/Netflix/p2plab"
)

type nodeSet struct {
}

func NewSet() p2plab.NodeSet {
	return &nodeSet{}
}

func (s *nodeSet) Add(n p2plab.Node) {
}

func (s *nodeSet) Remove(n p2plab.Node) {
}

func (s *nodeSet) Slice() []p2plab.Node {
	return nil
}

func (s *nodeSet) Label(ctx context.Context, addLabels, removeLabels []string) error {
	return nil
}
