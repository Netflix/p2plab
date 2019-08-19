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
	"context"

	"github.com/Netflix/p2plab"
)

func Execute(ctx context.Context, ls []p2plab.Labeled, q string) (p2plab.LabeledSet, error) {
	qry, err := Parse(q)
	if err != nil {
		return nil, err
	}

	lset := NewLabeledSet()
	for _, l := range ls {
		lset.Add(l)
	}

	mset, err := qry.Match(ctx, lset)
	if err != nil {
		return nil, err
	}

	return mset, nil
}
