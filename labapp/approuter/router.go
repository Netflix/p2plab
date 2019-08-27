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

package approuter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/Netflix/p2plab/daemon"
	"github.com/Netflix/p2plab/errdefs"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/peer"
	"github.com/Netflix/p2plab/pkg/logutil"
	cid "github.com/ipfs/go-cid"
	files "github.com/ipfs/go-ipfs-files"
	libp2ppeer "github.com/libp2p/go-libp2p-core/peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	multiaddr "github.com/multiformats/go-multiaddr"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type router struct {
	peer *peer.Peer
}

func New(p *peer.Peer) daemon.Router {
	return &router{p}
}

func (s *router) Routes() []daemon.Route {
	return []daemon.Route{
		// GET
		daemon.NewGetRoute("/peerInfo", s.getPeerInfo),
		daemon.NewGetRoute("/report", s.getReport),
		// POST
		daemon.NewPostRoute("/run", s.postRunTask),
	}
}

func (s *router) getPeerInfo(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	peerInfo := peerstore.PeerInfo{
		ID:    s.peer.Host().ID(),
		Addrs: s.peer.Host().Addrs(),
	}
	return daemon.WriteJSON(w, &peerInfo)
}

func (s *router) getReport(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	report, err := s.peer.Report(ctx)
	if err != nil {
		return err
	}
	return daemon.WriteJSON(w, &report)
}

func (s *router) postRunTask(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	var task metadata.Task
	err := json.NewDecoder(r.Body).Decode(&task)
	if err != nil {
		return err
	}

	ctx, logger := logutil.WithResponseLogger(ctx, w)
	logger.UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.Str("task", string(task.Type)).Str("subject", task.Subject)
	})

	switch task.Type {
	case metadata.TaskGet:
		err = s.getFile(ctx, task.Subject)
	case metadata.TaskConnect:
		addrs := strings.Split(task.Subject, ",")
		err = s.connect(ctx, addrs)
	case metadata.TaskDisconnect:
		addrs := strings.Split(task.Subject, ",")
		err = s.disconnect(ctx, addrs)
	default:
		return errors.Wrapf(errdefs.ErrInvalidArgument, "unrecognized task type: %q", task.Type)
	}
	if err != nil {
		return err
	}

	return nil
}

func (s *router) getFile(ctx context.Context, target string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "get")
	defer span.Finish()
	span.SetTag("cid", target)

	c, err := cid.Parse(target)
	if err != nil {
		return errors.Wrapf(errdefs.ErrInvalidArgument, "%s", err)
	}

	nd, err := s.peer.Get(ctx, c)
	if err != nil {
		return errors.Wrap(err, "failed to get file")
	}
	defer nd.Close()

	piper, pipew := io.Pipe()
	defer piper.Close()

	w, err := files.NewTarWriter(pipew)
	if err != nil {
		return errors.Wrap(err, "failed to create tar writer")
	}

	go func() {
		err := w.WriteFile(nd, c.String())
		if err != nil {
			pipew.CloseWithError(err)
			return
		}
		w.Close()
		pipew.Close()
	}()

	n, err := io.Copy(ioutil.Discard, piper)
	if err != nil {
		return err
	}

	zerolog.Ctx(ctx).Debug().Str("cid", c.String()).Int64("bytes", n).Msg("Retrieved file")
	return nil
}

func (s *router) connect(ctx context.Context, addrs []string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "connect")
	defer span.Finish()
	span.SetTag("addrs", fmt.Sprintf("%s", addrs))

	infos, err := parseAddrs(addrs)
	if err != nil {
		return err
	}

	err = s.peer.Connect(ctx, infos)
	if err != nil {
		return err
	}

	zerolog.Ctx(ctx).Debug().Int("peers", len(addrs)).Msg("Connected to peers")
	return nil
}

func (s *router) disconnect(ctx context.Context, addrs []string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "disconnect")
	defer span.Finish()
	span.SetTag("addrs", fmt.Sprintf("%s", addrs))

	infos, err := parseAddrs(addrs)
	if err != nil {
		return err
	}

	err = s.peer.Disconnect(ctx, infos)
	if err != nil {
		return err
	}

	zerolog.Ctx(ctx).Debug().Int("peers", len(addrs)).Msg("Disconnected from peers")
	return nil
}

func parseAddrs(addrs []string) ([]libp2ppeer.AddrInfo, error) {
	var infos []libp2ppeer.AddrInfo
	for _, addr := range addrs {
		ma, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			return nil, err
		}

		info, err := libp2ppeer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			return nil, err
		}

		infos = append(infos, *info)
	}
	return infos, nil
}
