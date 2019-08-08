package p2plab

import (
	"context"
	"io"

	blockservice "github.com/ipfs/go-blockservice"
	cid "github.com/ipfs/go-cid"
	datastore "github.com/ipfs/go-datastore"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	provider "github.com/ipfs/go-ipfs-provider"
	ipld "github.com/ipfs/go-ipld-format"
	ufsio "github.com/ipfs/go-unixfs/io"
	host "github.com/libp2p/go-libp2p-core/host"
	routing "github.com/libp2p/go-libp2p-routing"
)

type Peer interface {
	Host() host.Host
	DAGService() ipld.DAGService
	Provider() provider.System
	ContentRouting() routing.ContentRouting
	BlockService() blockservice.BlockService
	Blockstore() blockstore.Blockstore
	Datastore() datastore.Batching

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
