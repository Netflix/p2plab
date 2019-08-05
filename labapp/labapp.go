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
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"syscall"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/labagent"
	"github.com/Netflix/p2plab/peer"
	"github.com/containerd/fifo"
	cid "github.com/ipfs/go-cid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type LabApp struct {
	root   string
	fifoReader string
	fifoWriter string
	peer   *peer.Peer

	taskEncoder *json.Encoder
	taskDecoder *json.Decoder
}

func New(root, fifoReader, fifoWriter string) *LabApp {
	return &LabApp{
		root:   root,
		fifoReader: fifoReader,
		fifoWriter: fifoWriter,
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

	fifoReader, err := fifo.OpenFifo(ctx, a.fifoReader, syscall.O_RDONLY, 0700)
	if err != nil {
		return errors.Wrap(err, "failed to open fifo")
	}
	defer fifoReader.Close()

	fifoWriter, err := fifo.OpenFifo(ctx, a.fifoWriter, syscall.O_WRONLY, 0700)
	if err != nil {
		return errors.Wrap(err, "failed to open fifo")
	}
	defer fifoWriter.Close()

	a.taskEncoder, a.taskDecoder = json.NewEncoder(fifoWriter), json.NewDecoder(fifoReader)
	go a.handleTasks(ctx)

	var addrs []string
	for _, ma := range a.peer.Host.Addrs() {
		addrs = append(addrs, ma.String())
	}
	log.Info().Msgf("Listening on %s", addrs)

	return a.peer.Run()
}

func (a *LabApp) handleTasks(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			var task p2plab.Task
			err := a.taskDecoder.Decode(&task)
			if err != nil {
				log.Warn().Msgf("failed to decode request: %s", err)
				continue
			}

			var taskErr error
			ctx := context.Background()
			switch task.Type {
			case p2plab.TaskGetDAG:
				taskErr = a.handleTaskGet(ctx, task.Target)
			default:
				taskErr = errors.Errorf("unrecognized task type %q", task.Type)
			}

			resp := labagent.TaskResponse{
				Err: taskErr.Error(),
			}

			err = a.taskEncoder.Encode(&resp)
			if err != nil {
				log.Warn().Msgf("failed to encode response: %s", err)
			}
		}
	}
}

func (a *LabApp) handleTaskGet(ctx context.Context, target string) error {
	targetCid, err := cid.Parse(target)
	if err != nil {
		return err
	}

	r, err := a.peer.GetFile(ctx, targetCid)
	if err != nil {
		return err
	}
	defer r.Close()

	buf := new(bytes.Buffer)
	teeReader := io.TeeReader(r, buf)

	_, err = io.Copy(ioutil.Discard, teeReader)
	if err != nil {
		return err
	}

	return nil
}
