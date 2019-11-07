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

package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/peer"
	cid "github.com/ipfs/go-cid"
	"github.com/ipfs/go-graphsync/ipldbridge"
	"github.com/ipfs/go-graphsync/testutil"
	ipld "github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	mh "github.com/multiformats/go-multihash"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	// UNIX Time is faster and smaller than most timestamps. If you set
	// zerolog.TimeFieldFormat to an empty string, logs will write with UNIX
	// time.
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

func main() {
	err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "blockchain: %s\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	root := "./tmp/ociadd"
	err := os.MkdirAll(root, 0711)
	if err != nil {
		return err
	}

	p, err := peer.New(ctx, filepath.Join(root, "peer"), 0, metadata.PeerDefinition{
		Transports:         []string{"tcp"},
		Muxers:             []string{"mplex"},
		SecurityTransports: []string{"secio"},
		Routing:            "nil",
	})
	if err != nil {
		return err
	}

	var addrs []string
	for _, ma := range p.Host().Addrs() {
		addrs = append(addrs, ma.String())
	}
	log.Info().Str("id", p.Host().ID().String()).Strs("listen", addrs).Msg("Starting libp2p peer")

	bc, err := setupBlockChain(ctx, p.IPLDStorer(), p.IPLDBridge(), 100, 100)
	if err != nil {
		return err
	}

	c, err := cid.Parse(bc.tipLink.String())
	if err != nil {
		return err
	}

	log.Info().Str("tip", c.String()).Msg("Added blockchain")

	log.Info().Msgf("Retrieve manifest from another p2plab/peer by running:\n\ngo run ./cmd/ociget %s/p2p/%s %s\n", p.Host().Addrs()[0], p.Host().ID(), c)

	log.Info().Msgf("Connect to this peer from IPFS daemon:\n\nipfs swarm connect %s/p2p/%s\nipfs pin add %s\n", p.Host().Addrs()[0], p.Host().ID(), c)

	fmt.Print("Press 'Enter' to terminate peer...")
	_, err = bufio.NewReader(os.Stdin).ReadBytes('\n')
	if err != nil {
		return err
	}

	return nil
}

type blockChain struct {
	genisisNode ipld.Node
	genisisLink ipld.Link
	middleNodes []ipld.Node
	middleLinks []ipld.Link
	tipNode     ipld.Node
	tipLink     ipld.Link
}

func createBlock(nb ipldbridge.NodeBuilder, parents []ipld.Link, size int64) ipld.Node {
	return nb.CreateMap(func(mb ipldbridge.MapBuilder, knb ipldbridge.NodeBuilder, vnb ipldbridge.NodeBuilder) {
		mb.Insert(knb.CreateString("Parents"), vnb.CreateList(func(lb ipldbridge.ListBuilder, vnb ipldbridge.NodeBuilder) {
			for _, parent := range parents {
				lb.Append(vnb.CreateLink(parent))
			}
		}))
		mb.Insert(knb.CreateString("Messages"), vnb.CreateList(func(lb ipldbridge.ListBuilder, vnb ipldbridge.NodeBuilder) {
			lb.Append(vnb.CreateBytes(testutil.RandomBytes(size)))
		}))
	})
}

func setupBlockChain(ctx context.Context, storer ipldbridge.Storer, bridge ipldbridge.IPLDBridge, size int64, blockChainLength int) (*blockChain, error) {
	linkBuilder := cidlink.LinkBuilder{Prefix: cid.NewPrefixV1(cid.DagCBOR, mh.SHA2_256)}
	genisisNode, err := bridge.BuildNode(func(nb ipldbridge.NodeBuilder) ipld.Node {
		return createBlock(nb, []ipld.Link{}, size)
	})
	if err != nil {
		return nil, err
	}

	genesisLink, err := linkBuilder.Build(ctx, ipldbridge.LinkContext{}, genisisNode, storer)
	if err != nil {
		return nil, err
	}

	parent := genesisLink
	middleNodes := make([]ipld.Node, 0, blockChainLength-2)
	middleLinks := make([]ipld.Link, 0, blockChainLength-2)
	for i := 0; i < blockChainLength-2; i++ {
		node, err := bridge.BuildNode(func(nb ipldbridge.NodeBuilder) ipld.Node {
			return createBlock(nb, []ipld.Link{parent}, size)
		})
		if err != nil {
			return nil, err
		}

		middleNodes = append(middleNodes, node)
		link, err := linkBuilder.Build(ctx, ipldbridge.LinkContext{}, node, storer)
		if err != nil {
			return nil, err
		}

		middleLinks = append(middleLinks, link)
		parent = link
	}

	tipNode, err := bridge.BuildNode(func(nb ipldbridge.NodeBuilder) ipld.Node {
		return createBlock(nb, []ipld.Link{parent}, size)
	})
	if err != nil {
		return nil, err
	}

	tipLink, err := linkBuilder.Build(ctx, ipldbridge.LinkContext{}, tipNode, storer)
	if err != nil {
		return nil, err
	}

	return &blockChain{genisisNode, genesisLink, middleNodes, middleLinks, tipNode, tipLink}, nil
}
