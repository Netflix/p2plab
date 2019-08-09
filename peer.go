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
	ipld "github.com/ipfs/go-ipld-format"
	ufsio "github.com/ipfs/go-unixfs/io"
)

type PeerProvider interface {
	CreatePeerGroup(ctx context.Context, id string, cdef metadata.ClusterDefinition) (*PeerGroup, error)

	DestroyPeerGroup(ctx context.Context, pg *PeerGroup) error
}

type PeerGroup struct {
	ID    string
	Peers []Peer
}

type Peer interface {
	Add(ctx context.Context, r io.Reader, opts ...AddOption) (ipld.Node, error)

	Get(ctx context.Context, c cid.Cid) (ufsio.ReadSeekCloser, error)
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
}
