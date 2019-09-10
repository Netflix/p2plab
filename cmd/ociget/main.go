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
	files "github.com/ipfs/go-ipfs-files"
	libp2ppeer "github.com/libp2p/go-libp2p-core/peer"
	multiaddr "github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
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
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "ociget: must specify peer addr and cid")
		os.Exit(1)
	}

	err := run(os.Args[1], os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "ociget: %s\n", err)
		os.Exit(1)
	}
}

func run(addr, ref string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	root := "./tmp/ociget"
	err := os.MkdirAll(root, 0711)
	if err != nil {
		return err
	}

	p, err := peer.New(ctx, filepath.Join(root, "peer"), 0, metadata.PeerDefinition{
		Transports:         []string{"tcp", "ws", "quic"},
		Muxers:             []string{"mplex", "yamux"},
		SecurityTransports: []string{"tls", "secio", "noise"},
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

	targetAddr, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return err
	}

	targetInfo, err := libp2ppeer.AddrInfoFromP2pAddr(targetAddr)
	if err != nil {
		return err
	}

	err = p.Host().Connect(ctx, *targetInfo)
	if err != nil {
		return err
	}
	log.Info().Str("addr", targetAddr.String()).Msg("Connected to peer")

	c, err := cid.Parse(ref)
	if err != nil {
		return err
	}

	nd, err := p.Get(ctx, c)
	if err != nil {
		return err
	}

	f, ok := nd.(files.Directory)
	if !ok {
		return errors.Errorf("expected %q to be a directory", c)
	}

	iter := f.Entries()
	for iter.Next() {
		log.Info().Str("name", iter.Name()).Msg("Found file in unixfs directory")
	}

	if iter.Err() != nil {
		return err
	}

	fmt.Print("Press 'Enter' to terminate peer...")
	_, err = bufio.NewReader(os.Stdin).ReadBytes('\n')
	if err != nil {
		return err
	}

	return nil
}
