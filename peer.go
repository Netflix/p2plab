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
	files "github.com/ipfs/go-ipfs-files"
	ipld "github.com/ipfs/go-ipld-format"
	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
)

// Peer is a minimal IPFS node that can distribute IPFS DAGs.
type Peer interface {
	// Host returns the libp2p host.
	Host() host.Host

	// DAGService returns the IPLD DAG service.
	DAGService() ipld.DAGService

	// Connect connects to the libp2p peers.
	Connect(ctx context.Context, infos []peer.AddrInfo) error

	// Disconnect disconnects from libp2p peers.
	Disconnect(ctx context.Context, infos []peer.AddrInfo) error

	// Add adds content from an io.Reader into the Peer's storage.
	Add(ctx context.Context, r io.Reader, opts ...AddOption) (ipld.Node, error)

	// Get returns an Unixfsv1 file from a given cid.
	Get(ctx context.Context, c cid.Cid) (files.Node, error)

	// FetchGraph fetches the full DAG rooted at a given cid.
	FetchGraph(ctx context.Context, c cid.Cid) error

	// Report returns all the metrics collected from the peer.
	Report(ctx context.Context) (metadata.ReportNode, error)
}

// AddOption is an option for AddSettings.
type AddOption func(*AddSettings) error

// AddSettings describe the settings for adding content to the peer.
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

// WithLayout sets the format for DAG generation.
func WithLayout(layout string) AddOption {
	return func(s *AddSettings) error {
		s.Layout = layout
		return nil
	}
}

// WithChunker sets the chunking strategy for the content.
func WithChunker(chunker string) AddOption {
	return func(s *AddSettings) error {
		s.Chunker = chunker
		return nil
	}
}

// WithRawLeaves sets whether to use raw blocks for leaf nodes.
func WithRawLeaves(rawLeaves bool) AddOption {
	return func(s *AddSettings) error {
		s.RawLeaves = rawLeaves
		return nil
	}
}

// WithHashFunc sets the hashing function for the blocks.
func WithHashFunc(hashFunc string) AddOption {
	return func(s *AddSettings) error {
		s.HashFunc = hashFunc
		return nil
	}
}

// WithMaxLinks sets the maximum children each block can have.
func WithMaxLinks(maxLinks int) AddOption {
	return func(s *AddSettings) error {
		s.MaxLinks = maxLinks
		return nil
	}
}
