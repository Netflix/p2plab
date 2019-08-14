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

	cid "github.com/ipfs/go-cid"
	files "github.com/ipfs/go-ipfs-files"
	ipld "github.com/ipfs/go-ipld-format"
	"github.com/libp2p/go-libp2p-core/peer"
	host "github.com/libp2p/go-libp2p-core/host"
)

type Peer interface {
	Host() host.Host

	DAGService() ipld.DAGService

	Connect(ctx context.Context, infos []peer.AddrInfo) error

	Disconnect(ctx context.Context, infos []peer.AddrInfo) error

	Add(ctx context.Context, r io.Reader, opts ...AddOption) (ipld.Node, error)

	Get(ctx context.Context, c cid.Cid) (files.Node, error)
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
