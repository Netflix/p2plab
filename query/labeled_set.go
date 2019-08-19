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
	"sort"

	"github.com/Netflix/p2plab"
)

type labeledSet struct {
	set map[string]p2plab.Labeled
}

func NewLabeledSet() p2plab.LabeledSet {
	return &labeledSet{
		set: make(map[string]p2plab.Labeled),
	}
}

func (s *labeledSet) Add(l p2plab.Labeled) {
	s.set[l.ID()] = l
}

func (s *labeledSet) Remove(id string) {
	delete(s.set, id)
}

func (s *labeledSet) Get(id string) p2plab.Labeled {
	return s.set[id]
}

func (s *labeledSet) Contains(id string) bool {
	return s.Get(id) != nil
}

func (s *labeledSet) Slice() []p2plab.Labeled {
	var slice []p2plab.Labeled
	for _, l := range s.set {
		slice = append(slice, l)
	}
	sort.SliceStable(slice, func(i, j int) bool {
		return slice[i].ID() < slice[j].ID()
	})
	return slice
}
