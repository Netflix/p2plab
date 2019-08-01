// Copyright 2019 Netflix, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package nodes

import (
	"context"
	"testing"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/metadata"
	"github.com/stretchr/testify/require"
)

type testNode struct {
	id string
}

func (n *testNode) Metadata() metadata.Node {
	return metadata.Node{ID: n.id}
}

func (n *testNode) SSH(ctx context.Context, opts ...p2plab.SSHOption) error {
	return nil
}

func newNode(id string) p2plab.Node {
	return &testNode{id}
}

func TestNodeSetAdd(t *testing.T) {
	// Empty set starts with 0 elements.
	nset := NewSet()
	require.Empty(t, nset.Slice())

	// Adding node increases length by 1.
	nset.Add(newNode("1"))
	require.Len(t, nset.Slice(), 1)

	// Adding duplicate node does nothing.
	nset.Add(newNode("1"))
	require.Len(t, nset.Slice(), 1)
}

func TestNodeSetRemove(t *testing.T) {
	nset := NewSet()
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
