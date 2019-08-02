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

package labapp

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"syscall"

	"github.com/Netflix/p2plab/labagent"
	"github.com/Netflix/p2plab/peer"
	"github.com/containerd/fifo"
	cid "github.com/ipfs/go-cid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type LabApp struct {
	root   string
	fdPath string
	peer   *peer.Peer
}

func New(root, fdPath string) *LabApp {
	return &LabApp{
		root:   root,
		fdPath: fdPath,
	}
}

func (a *LabApp) Serve(ctx context.Context) error {
	ds, err := peer.NewDatastore(a.root)
	if err != nil {
		return errors.Wrap(err, "failed to create datastore")
	}

	host, r, err := peer.NewLibp2pPeer(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to create libp2p peer")
	}

	a.peer, err = peer.NewPeer(ctx, ds, host, r)
	if err != nil {
		return errors.Wrap(err, "failed to create peer peer")
	}

	f, err := fifo.OpenFifo(ctx, a.fdPath, syscall.O_RDONLY, 0700)
	if err != nil {
		return errors.Wrap(err, "failed to open fifo")
	}
	defer f.Close()

	done := make(chan struct{})
	defer func() {
		<-done
	}()

	encoder, decoder := json.NewEncoder(f), json.NewDecoder(f)
	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				var req labagent.TaskRequest
				err = decoder.Decode(&req)
				if err != nil {
					log.Warn().Msgf("failed to decode request: %s", err)
					continue
				}

				ctx := context.Background()
				var taskErr error
				switch req.Type {
				case labagent.TaskGet:
					if len(req.Args) != 1 {
						taskErr = errors.Errorf("get must have exactly 1 arg")
						break
					}

					target, err := cid.Parse(req.Args[0])
					if err != nil {
						taskErr = err
						break
					}

					var r io.ReadCloser
					r, taskErr = a.peer.GetFile(ctx, target)
					if taskErr == nil {
						defer r.Close()

						_, err = io.Copy(ioutil.Discard, r)
						if err != nil {
							taskErr = err
						}
					}
				default:
					taskErr = errors.Errorf("unrecognized task type %q", req.Type)
				}

				resp := labagent.TaskResponse{
					Err: taskErr,
				}

				err = encoder.Encode(&resp)
				if err != nil {
					log.Warn().Msgf("failed to encode response: %s", err)
				}
			}
		}
	}()

	var addrs []string
	for _, ma := range a.peer.Host.Addrs() {
		addrs = append(addrs, ma.String())
	}
	log.Info().Msgf("Listening on %s", addrs)

	return a.peer.Run()
}
