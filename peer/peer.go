package peer

import (
	"context"
	"io"
	"strings"
	"time"

	bitswap "github.com/ipfs/go-bitswap"
	"github.com/ipfs/go-bitswap/network"
	blockservice "github.com/ipfs/go-blockservice"
	cid "github.com/ipfs/go-cid"
	datastore "github.com/ipfs/go-datastore"
	badger "github.com/ipfs/go-ds-badger"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	chunker "github.com/ipfs/go-ipfs-chunker"
	provider "github.com/ipfs/go-ipfs-provider"
	"github.com/ipfs/go-ipfs-provider/queue"
	"github.com/ipfs/go-ipfs-provider/simple"
	cbor "github.com/ipfs/go-ipld-cbor"
	ipld "github.com/ipfs/go-ipld-format"
	dag "github.com/ipfs/go-merkledag"
	"github.com/ipfs/go-unixfs/importer/balanced"
	"github.com/ipfs/go-unixfs/importer/helpers"
	"github.com/ipfs/go-unixfs/importer/trickle"
	ufsio "github.com/ipfs/go-unixfs/io"
	libp2p "github.com/libp2p/go-libp2p"
	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/routing"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	mplex "github.com/libp2p/go-libp2p-mplex"
	secio "github.com/libp2p/go-libp2p-secio"
	tcp "github.com/libp2p/go-tcp-transport"
	ws "github.com/libp2p/go-ws-transport"
	multihash "github.com/multiformats/go-multihash"
	"github.com/pkg/errors"
)

func init() {
	ipld.Register(cid.DagProtobuf, dag.DecodeProtobufBlock)
	ipld.Register(cid.Raw, dag.DecodeRawBlock)
	ipld.Register(cid.DagCBOR, cbor.DecodeBlock) // need to decode CBOR
}

var (
	defaultReprovideInterval = 12 * time.Hour
)

type Peer struct {
	ipld.DAGService
	Host      host.Host

	ctx    context.Context
	r      routing.ContentRouting
	ds     datastore.Batching
	bs     blockstore.Blockstore
	bserv  blockservice.BlockService
	system provider.System
}

func NewPeer(ctx context.Context, ds datastore.Batching, h host.Host, r routing.ContentRouting) (*Peer, error) {
	bs, err := NewBlockstore(ctx, ds)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create blockstore")
	}

	bserv := NewBlockService(ctx, bs, h, r)

	system, err := NewProviderSystem(ctx, ds, bs, r)
	if err != nil {
		bserv.Close()
		return nil, errors.Wrap(err, "failed to create provider system")
	}

	return &Peer{
		ctx:        ctx,
		DAGService: dag.NewDAGService(bserv),
		Host:          h,
		r:          r,
		ds:         ds,
		bs:         bs,
		bserv:      bserv,
		system:     system,
	}, nil
}

func (p *Peer) Run() error {
	p.system.Run()

	select {
	case <-p.ctx.Done():
		err := p.system.Close()
		if err != nil {
			return err
		}
		return p.bserv.Close()
	}
}

// AddParams contains all of the configurable parameters needed to specify the
// importing process of a file.
type AddParams struct {
	Layout    string
	Chunker   string
	RawLeaves bool
	Hidden    bool
	Shard     bool
	NoCopy    bool
	HashFunc  string
}

// AddFile chunks and adds content to the DAGService from a reader. The content
// is stored as a UnixFS DAG (default for IPFS). It returns the root
// ipld.Node.
func (p *Peer) AddFile(r io.Reader, params *AddParams) (ipld.Node, error) {
	if params == nil {
		params = &AddParams{}
	}
	if params.HashFunc == "" {
		params.HashFunc = "sha2-256"
	}

	prefix, err := dag.PrefixForCidVersion(1)
	if err != nil {
		return nil, errors.Wrap(err, "unrecognized CID version")
	}

	hashFuncCode, ok := multihash.Names[strings.ToLower(params.HashFunc)]
	if !ok {
		return nil, errors.Wrapf(err, "unrecognized hash function %q", params.HashFunc)
	}
	prefix.MhType = hashFuncCode

	dbp := helpers.DagBuilderParams{
		Dagserv:    p,
		RawLeaves:  params.RawLeaves,
		Maxlinks:   helpers.DefaultLinksPerBlock,
		NoCopy:     params.NoCopy,
		CidBuilder: &prefix,
	}

	chnk, err := chunker.FromString(r, params.Chunker)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create chunker")
	}

	dbh, err := dbp.New(chnk)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create dag builder")
	}

	var n ipld.Node
	switch params.Layout {
	case "trickle":
		n, err = trickle.Layout(dbh)
	case "balanced", "":
		n, err = balanced.Layout(dbh)
	default:
		return nil, errors.Errorf("unrecognized layout %q", params.Layout)
	}

	return n, err
}

// GetFile returns a reader to a file as identified by its root CID. The file
// must have been added as a UnixFS DAG (default for IPFS).
func (p *Peer) GetFile(ctx context.Context, c cid.Cid) (ufsio.ReadSeekCloser, error) {
	n, err := p.Get(ctx, c)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get file %q", c)
	}
	return ufsio.NewDagReader(ctx, n, p)
}

func NewDatastore(path string) (datastore.Batching, error) {
	return badger.NewDatastore(path, &badger.DefaultOptions)
}

func NewLibp2pPeer(ctx context.Context) (host.Host, routing.ContentRouting, error) {
	transports := libp2p.ChainOptions(
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(ws.New),
	)

	muxers := libp2p.ChainOptions(
		libp2p.Muxer("/mplex/6.7.0", mplex.DefaultTransport),
	)

	security := libp2p.Security(secio.ID, secio.New)

	listenAddrs := libp2p.ListenAddrStrings(
		"/ip4/0.0.0.0/tcp/0",
	)

	var dht *kaddht.IpfsDHT
	newDHT := func(h host.Host) (routing.PeerRouting, error) {
		var err error
		dht, err = kaddht.New(ctx, h)
		return dht, err
	}
	routing := libp2p.Routing(newDHT)

	host, err := libp2p.New(
		ctx,
		transports,
		listenAddrs,
		muxers,
		security,
		routing,
	)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create libp2p host")
	}

	return host, dht, nil
}

func NewBlockstore(ctx context.Context, ds datastore.Batching) (blockstore.Blockstore, error) {
	bs := blockstore.NewBlockstore(ds)
	bs = blockstore.NewIdStore(bs)

	// var err error
	// bs, err = blockstore.CachedBlockstore(ctx, bs, blockstore.DefaultCacheOpts())
	// if err != nil {
	// 	return nil, err
	// }

	return bs, nil
}

func NewBlockService(ctx context.Context, bs blockstore.Blockstore, h host.Host, r routing.ContentRouting) blockservice.BlockService {
	bswapnet := network.NewFromIpfsHost(h, r)
	rem := bitswap.New(ctx, bswapnet, bs)
	return blockservice.New(bs, rem)
}

func NewProviderSystem(ctx context.Context, ds datastore.Batching, bs blockstore.Blockstore, r routing.ContentRouting) (provider.System, error) {
	queue, err := queue.NewQueue(ctx, "repro", ds)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new queue")
	}

	prov := simple.NewProvider(ctx, queue, r)
	reprov := simple.NewReprovider(ctx, 12*time.Hour, r, simple.NewBlockstoreProvider(bs))
	system := provider.NewSystem(prov, reprov)
	return system, nil
}
