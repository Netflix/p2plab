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

package labd

import (
	"context"

	"github.com/Netflix/p2plab"
)

type nodeSet struct {
	cln *client
}

func (napi *nodeAPI) NewSet() p2plab.NodeSet {
	return &nodeSet{napi.cln}
}

func (s *nodeSet) Add(n p2plab.Node) {
}

func (s *nodeSet) Remove(n p2plab.Node) {
}

func (s *nodeSet) Slice() []p2plab.Node {
	return nil
}

func (s *nodeSet) Label(ctx context.Context, addLabels, removeLabels []string) error {
	req := s.cln.NewRequest("PUT", "/nodes")
	resp, err := req.Send(ctx)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
