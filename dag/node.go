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
	"io"

	chunk "github.com/ipfs/go-ipfs-chunker"
	ipld "github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/fluent"
)

type Builder struct {
	nb     fluent.NodeBuilder
	lb     ipld.LinkBuilder
	storer ipld.Storer
}

func NewBuilder(nb ipld.NodeBuilder, lb ipld.LinkBuilder, storer ipld.Storer) *Builder {
	return &Builder{
		nb:     fluent.WrapNodeBuilder(nb),
		lb:     lb,
		storer: storer,
	}
}

func (b *Builder) NewNode(data []byte, links []*Link) (ipld.Node, error) {
	var nd ipld.Node
	err := fluent.Recover(func() {
		nd = b.nb.CreateMap(func(mb fluent.MapBuilder, knb fluent.NodeBuilder, vnb fluent.NodeBuilder) {
			mb.Insert(knb.CreateString("data"), vnb.CreateBytes(data))
			mb.Insert(knb.CreateString("links"), vnb.CreateList(func(lb fluent.ListBuilder, vnb fluent.NodeBuilder) {
				for _, link := range links {
					lb.Append(vnb.CreateMap(func(mb fluent.MapBuilder, knb fluent.NodeBuilder, vnb fluent.NodeBuilder) {
						mb.Insert(knb.CreateString("Cid"), vnb.CreateLink(link.Link))
						mb.Insert(knb.CreateString("Name"), vnb.CreateString(link.Name))
						mb.Insert(knb.CreateString("Size"), vnb.CreateInt(link.Size))
					}))
				}
			}))
		})
	})
	return nd, err
}

func (b *Builder) Build(ctx context.Context, splitter chunk.Splitter) (ipld.Link, error) {
	var links []*Link

	chunk, err := splitter.NextBytes()
	for err == nil {
		nd, err := b.NewNode(chunk, []*Link{})
		if err != nil {
			return nil, err
		}

		link := &Link{Size: len(chunk)}
		link.Link, err = b.lb.Build(ctx, ipld.LinkContext{}, nd, b.storer)
		if err != nil {
			return nil, err
		}
		links = append(links, link)

		chunk, err = splitter.NextBytes()
		if err == io.EOF {
			err = nil
			break
		}
	}
	if err != nil {
		return nil, err
	}

	nd, err := b.NewNode(nil, links)
	if err != nil {
		return nil, err
	}

	return b.lb.Build(ctx, ipld.LinkContext{}, nd, b.storer)
}

type Link struct {
	Link ipld.Link
	Name string
	Size int
}
