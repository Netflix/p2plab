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

	"context"

	"github.com/Netflix/p2plab"
	"github.com/stretchr/testify/require"
)

func GetLabeled() []p2plab.Labeled {
	return []p2plab.Labeled{
		NewLabeled("apple", []string{"everyone", "apple", "slowdisk", "region=us-west-2"}),
		NewLabeled("banana", []string{"everyone", "banana", "region=us-west-2"}),
		NewLabeled("cherry", []string{"everyone", "cherry", "region=us-east-1"}),
	}
}

func TestExecute(t *testing.T) {
	ctx := context.TODO()
	ls := GetLabeled()

	labeledSet, error := Execute(ctx, ls, "apple")
	if error != nil {
		return
	}
	require.Equal(t, []p2plab.Labeled{ls[0]}, labeledSet.Slice(), "Query apple return only apple node")

	labeledSet, error = Execute(ctx, ls, "(not ‘apple’)")
	if error != nil {
		return
	}
	require.Equal(t, []p2plab.Labeled{ls[1], ls[2]}, labeledSet.Slice(), "Query (not ‘apple’) return banana and cherry node")

	labeledSet, error = Execute(ctx, ls, "(and ‘slowdisk’ ‘region=us-west-2’)")
	if error != nil {
		return
	}
	require.Equal(t, []p2plab.Labeled{ls[0]}, labeledSet.Slice(), "Query (and ‘slowdisk’ ‘region=us-west-2’) return only apple node")

	labeledSet, error = Execute(ctx, ls, "(or ‘region=us-west-2’ ‘region=us-east-1’)")
	if error != nil {
		return
	}
	require.Equal(t, ls, labeledSet.Slice(), "Query (or ‘region=us-west-2’ ‘region=us-east-1’) return apple, banana, cherry node")

	labeledSet, error = Execute(ctx, ls, "(or (not ‘slowdisk’) ‘banana’)")
	if error != nil {
		return
	}
	require.Equal(t, []p2plab.Labeled{ls[1], ls[2]}, labeledSet.Slice(), "Query (or (not ‘slowdisk’) ‘banana’) return banana and cherry node")
}
