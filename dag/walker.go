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

package approuter

import (
	"context"

	cid "github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"
)

func Walk(ctx context.Context, c cid.Cid, dserv ipld.DAGService) error {
	nd, err := dserv.Get(ctx, c)
	if err != nil {
		return err
	}

	var cids []cid.Cid
	for _, link := range nd.Links() {
		cids = append(cids, link.Cid)
	}

	eg, gctx := errgroup.WithContext(ctx)

	ndChan := dserv.GetMany(ctx, cids)
	for ndOpt := range ndChan {
		if ndOpt.Err != nil {
			return err
		}

		child := ndOpt.Node.Cid()
		eg.Go(func() {
			return Walk(ctx, child, dserv)
		})
	}

	err = eg.Wait()
	if err != nil {
		return err
	}

	return nil
}
