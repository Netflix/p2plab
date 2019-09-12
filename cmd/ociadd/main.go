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
	"github.com/Netflix/p2plab/pkg/httputil"
	"github.com/Netflix/p2plab/transformers/oci"
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
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "ociadd: must specify ref")
		os.Exit(1)
	}

	err := run(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "ociadd: %s\n", err)
		os.Exit(1)
	}
}

func run(ref string) error {
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

	transformer, err := oci.New(filepath.Join(root, "transformers/oci"), httputil.NewHTTPClient())
	if err != nil {
		return err
	}

	c, err := transformer.Transform(ctx, p, ref)
	if err != nil {
		return err
	}
	log.Info().Str("ref", ref).Str("cid", c.String()).Msg("Converted OCI image to IPLD DAG")

	log.Info().Msgf("Retrieve manifest from another p2plab/peer by running:\n\ngo run ./cmd/ociget %s/p2p/%s %s\n", p.Host().Addrs()[0], p.Host().ID(), c)

	log.Info().Msgf("Connect to this peer from IPFS daemon:\n\nipfs swarm connect %s/p2p/%s\nipfs cat %s\n", p.Host().Addrs()[0], p.Host().ID(), c)

	fmt.Print("Press 'Enter' to terminate peer...")
	_, err = bufio.NewReader(os.Stdin).ReadBytes('\n')
	if err != nil {
		return err
	}

	return nil
}
