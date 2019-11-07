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

package p2plab

import (
	"context"
	"io"

	"github.com/Netflix/p2plab/metadata"
	cid "github.com/ipfs/go-cid"
	"github.com/ipfs/go-graphsync/ipldbridge"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	files "github.com/ipfs/go-ipfs-files"
	format "github.com/ipfs/go-ipld-format"
	ipld "github.com/ipld/go-ipld-prime"
	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
)

type Peer interface {
	Host() host.Host

	DAGService() format.DAGService

	Connect(ctx context.Context, infos []peer.AddrInfo) error

	Disconnect(ctx context.Context, infos []peer.AddrInfo) error

	Add(ctx context.Context, r io.Reader, opts ...AddOption) (format.Node, error)

	Get(ctx context.Context, c cid.Cid) (files.Node, error)

	FetchGraph(ctx context.Context, c cid.Cid) error

	Report(ctx context.Context) (metadata.ReportNode, error)

	// GraphSync

	Blockstore() blockstore.Blockstore

	IPLDStorer() ipldbridge.Storer

	IPLDBridge() ipldbridge.IPLDBridge

	AddPrime(ctx context.Context, r io.Reader, opts ...AddOption) (ipld.Link, error)

	GraphSync(ctx context.Context, c cid.Cid, targetPeer peer.ID) error
}

type AddOption func(*AddSettings) error

type AddSettings struct {
	Layout    string
	Chunker   string
	RawLeaves bool
	Hidden    bool
	Shard     bool
	NoCopy    bool
	HashFunc  string
	MaxLinks  int
}

func WithLayout(layout string) AddOption {
	return func(s *AddSettings) error {
		s.Layout = layout
		return nil
	}
}

func WithChunker(chunker string) AddOption {
	return func(s *AddSettings) error {
		s.Chunker = chunker
		return nil
	}
}

func WithRawLeaves(rawLeaves bool) AddOption {
	return func(s *AddSettings) error {
		s.RawLeaves = rawLeaves
		return nil
	}
}

func WithHashFunc(hashFunc string) AddOption {
	return func(s *AddSettings) error {
		s.HashFunc = hashFunc
		return nil
	}
}

func WithMaxLinks(maxLinks int) AddOption {
	return func(s *AddSettings) error {
		s.MaxLinks = maxLinks
		return nil
	}
}
