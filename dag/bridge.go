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

package dag

import (
	"context"

	"github.com/ipfs/go-graphsync/ipldbridge"
	ipld "github.com/ipld/go-ipld-prime"
	dagpb "github.com/ipld/go-ipld-prime-proto"
	free "github.com/ipld/go-ipld-prime/impl/free"
	"github.com/ipld/go-ipld-prime/traversal"
)

var (
	defaultChooser traversal.NodeBuilderChooser = dagpb.AddDagPBSupportToChooser(func(ipld.Link, ipld.LinkContext) ipld.NodeBuilder {
		return free.NodeBuilder()
	})
)

type ipldBridge struct {
	ipldbridge.IPLDBridge
}

// NewIPLDBridge returns an IPLD Bridge.
func NewIPLDBridge() ipldbridge.IPLDBridge {
	return &ipldBridge{ipldbridge.NewIPLDBridge()}
}

func (ib *ipldBridge) Traverse(ctx context.Context, loader ipldbridge.Loader, root ipld.Link, s ipldbridge.Selector, fn ipldbridge.AdvVisitFn) error {
	node, err := root.Load(ctx, ipldbridge.LinkContext{}, dagpb.PBNode__NodeBuilder(), loader)
	if err != nil {
		return err
	}
	return traversal.Progress{
		Cfg: &traversal.Config{
			Ctx:                    ctx,
			LinkLoader:             loader,
			LinkNodeBuilderChooser: defaultChooser,
		},
	}.WalkAdv(node, s, fn)
}
