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

	"github.com/Netflix/p2plab/pkg/traceutil"
	"github.com/hako/durafmt"
	cid "github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"
	merkledag "github.com/ipfs/go-merkledag"
	opentracing "github.com/opentracing/opentracing-go"
	"golang.org/x/sync/errgroup"
)

func Walk(ctx context.Context, c cid.Cid, ng ipld.NodeGetter) error {
	span, ctx := traceutil.StartSpanFromContext(ctx, "dag.Walk")
	defer span.Finish()
	span.SetTag("cid", c.String())

	defer func() {
		span.SetTag("culmIpldDecodeTime", durafmt.Parse(merkledag.CulmIpldDecodeTime).String())
		span.SetTag("culmWaitForIpldNodeOut", durafmt.Parse(merkledag.CulmWaitForIpldNodeOut).String())
	}()

	nd, err := ng.Get(ctx, c)
	if err != nil {
		return err
	}

	return walk(ctx, nd, ng)
}

func walk(ctx context.Context, nd ipld.Node, ng ipld.NodeGetter) error {
	var cids []cid.Cid
	for _, link := range nd.Links() {
		cids = append(cids, link.Cid)
	}

	if len(cids) > 0 {
		var span opentracing.Span
		span, ctx = traceutil.StartSpanFromContext(ctx, "dag.walk")
		defer span.Finish()
		span.SetTag("cid", nd.Cid().String())
	}

	eg, gctx := errgroup.WithContext(ctx)

	ndChan := ng.GetMany(ctx, cids)
	for ndOpt := range ndChan {
		if ndOpt.Err != nil {
			return ndOpt.Err
		}

		nd := ndOpt.Node
		eg.Go(func() error {
			return walk(gctx, nd, ng)
		})
	}

	err := eg.Wait()
	if err != nil {
		return err
	}

	return nil
}
