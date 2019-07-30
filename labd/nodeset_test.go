package labd

import (
	"testing"

	"github.com/Netflix/p2plab"
	"github.com/stretchr/testify/require"
)

func newNode(id string) p2plab.Node {
	return &node{
		id: id,
	}
}

func TestNodeSetAdd(t *testing.T) {
	cln, err := NewClient("")
	require.NoError(t, err)

	// Empty set starts with 0 elements.
	nset := cln.Node().NewSet()
	require.Empty(t, nset.Slice())

	// Adding node increases length by 1.
	nset.Add(newNode("1"))
	require.Len(t, nset.Slice(), 1)

	// Adding duplicate node does nothing.
	nset.Add(newNode("1"))
	require.Len(t, nset.Slice(), 1)
}

func TestNodeSetRemove(t *testing.T) {
	cln, err := NewClient("")
	require.NoError(t, err)

	nset := cln.Node().NewSet()
	nset.Add(newNode("1"))
	require.Len(t, nset.Slice(), 1)

	// Removing non-existing node does nothing.
	nset.Remove(newNode("2"))
	require.Len(t, nset.Slice(), 1)

	// Removing existing node decreases length by 1.
	nset.Remove(newNode("1"))
	require.Empty(t, nset.Slice())

	// Removing pre-existing node does nothing.
	nset.Remove(newNode("1"))
	require.Empty(t, nset.Slice())
}
