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

var ls = []p2plab.Labeled{
	NewLabeled("apple", []string{"everyone", "apple", "slowdisk", "region=us-west-2"}),
	NewLabeled("banana", []string{"everyone", "banana", "region=us-west-2"}),
	NewLabeled("cherry", []string{"everyone", "cherry", "region=us-east-1"}),
}

var executetest = []struct {
	in  string
	out []p2plab.Labeled
}{
	{"'apple'", []p2plab.Labeled{ls[0]}},
	{"(not 'apple')", []p2plab.Labeled{ls[1], ls[2]}},
	{"(and 'slowdisk' 'region=us-west-2')", []p2plab.Labeled{ls[0]}},
	{"(or 'region=us-west-2' 'region=us-east-1')", ls},
	{"(or (not 'slowdisk') 'banana')", []p2plab.Labeled{ls[1], ls[2]}},
}

func TestExecute(t *testing.T) {
	ctx := context.Background()

	for _, execute := range executetest {
		labeledSet, err := Execute(ctx, ls, execute.in)

		require.NoError(t, err)
		require.Equal(t, execute.out, labeledSet.Slice())
	}
}
