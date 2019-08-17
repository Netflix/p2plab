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
	"sort"

	"github.com/Netflix/p2plab"
)

type nodeSet struct {
	set map[string]p2plab.Node
}

func NewSet() p2plab.NodeSet {
	return &nodeSet{
		set: make(map[string]p2plab.Node),
	}
}

func (s *nodeSet) Add(n p2plab.Node) {
	s.set[n.Metadata().ID] = n
}

func (s *nodeSet) Remove(n p2plab.Node) {
	delete(s.set, n.Metadata().ID)
}

func (s *nodeSet) Get(id string) p2plab.Node {
	return s.set[id]
}

func (s *nodeSet) Contains(id string) bool {
	return s.Get(id) != nil
}

func (s *nodeSet) Slice() []p2plab.Node {
	var slice []p2plab.Node
	for _, n := range s.set {
		slice = append(slice, n)
	}
	sort.SliceStable(slice, func(i, j int) bool {
		return slice[i].Metadata().ID < slice[j].Metadata().ID
	})
	return slice
}
