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

package query

import (
	"testing"

	"github.com/Netflix/p2plab"
	"github.com/stretchr/testify/require"
)

type testLabeled struct {
	id string
}

func (l *testLabeled) ID() string {
	return l.id
}

func (l *testLabeled) Labels() []string {
	return nil
}

func newLabeled(id string) p2plab.Labeled {
	return &testLabeled{id}
}

func TestLabeledSetAdd(t *testing.T) {
	// Empty set starts with 0 elements.
	lset := NewLabeledSet()
	require.Empty(t, lset.Slice())

	// Adding node increases length by 1.
	lset.Add(newLabeled("1"))
	require.Len(t, lset.Slice(), 1)

	// Adding duplicate node does nothing.
	lset.Add(newLabeled("1"))
	require.Len(t, lset.Slice(), 1)
}

func TestLabeledSetRemove(t *testing.T) {
	lset := NewLabeledSet()
	lset.Add(newLabeled("1"))
	require.Len(t, lset.Slice(), 1)

	// Removing non-existing node does nothing.
	lset.Remove("2")
	require.Len(t, lset.Slice(), 1)

	// Removing existing node decreases length by 1.
	lset.Remove("1")
	require.Empty(t, lset.Slice())

	// Removing pre-existing node does nothing.
	lset.Remove("1")
	require.Empty(t, lset.Slice())
}
