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
	"net/http"
	"strings"
	"time"

	"github.com/Netflix/p2plab/errdefs"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/peer"
	"github.com/Netflix/p2plab/pkg/httputil"
	"github.com/gorilla/mux"
	cid "github.com/ipfs/go-cid"
	files "github.com/ipfs/go-ipfs-files"
	libp2ppeer "github.com/libp2p/go-libp2p-core/peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	multiaddr "github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type LabApp struct {
	root   string
	addr   string
	ready  bool
	router *mux.Router
	peer   *peer.Peer
}

func New(root, addr string) *LabApp {
	r := mux.NewRouter().UseEncodedPath().StrictSlash(true)
	app := &LabApp{
		root:   root,
		addr:   addr,
		router: r,
	}
	app.registerRoutes(r)
	return app
}

func (a *LabApp) Serve(ctx context.Context) error {
	var err error
	a.peer, err = peer.New(ctx, a.root)
	if err != nil {
		return errors.Wrap(err, "failed to create peer")
	}

	var addrs []string
	for _, ma := range a.peer.Host().Addrs() {
		addrs = append(addrs, ma.String())
	}
	zerolog.Ctx(ctx).Info().Msgf("IPFS listening on %s", addrs)

	s := &http.Server{
		Handler:     a.router,
		Addr:        a.addr,
		ReadTimeout: 10 * time.Second,
	}
	zerolog.Ctx(ctx).Info().Msgf("labapp listening on %s", a.addr)

	a.ready = true
	return s.ListenAndServe()
}

func (a *LabApp) registerRoutes(r *mux.Router) {
	api := r.PathPrefix("/api/v0").Subrouter()
	api.HandleFunc("/healthcheck", a.healthcheckHandler).Methods("GET")
	api.Handle("/peerInfo", httputil.ErrorHandler{a.peerInfoHandler}).Methods("GET")
	api.Handle("/run", httputil.ErrorHandler{a.runHandler}).Methods("POST")
}

func (a *LabApp) healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	if a.ready {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Healthy"))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Unhealthy"))
	}
}

func (a *LabApp) peerInfoHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("labapp/peerInfo")
	peerInfo := peerstore.PeerInfo{
		ID:    a.peer.Host().ID(),
		Addrs: a.peer.Host().Addrs(),
	}
	return httputil.WriteJSON(w, &peerInfo)
}

func (a *LabApp) runHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("labapp/run")

	var task metadata.Task
	err := json.NewDecoder(r.Body).Decode(&task)
	if err != nil {
		return err
	}

	ctx := r.Context()
	switch task.Type {
	case metadata.TaskGet:
		err = a.getFile(ctx, task.Subject)
	case metadata.TaskConnect:
		addrs := strings.Split(task.Subject, ",")
		err = a.connect(ctx, addrs)
	case metadata.TaskDisconnect:
		addrs := strings.Split(task.Subject, ",")
		err = a.disconnect(ctx, addrs)
	default:
		return errors.Wrapf(errdefs.ErrInvalidArgument, "unrecognized task type: %q", task.Type)
	}
	if err != nil {
		return err
	}

	return nil
}

func (a *LabApp) getFile(ctx context.Context, target string) error {
	c, err := cid.Parse(target)
	if err != nil {
		return errors.Wrap(err, "failed to parse cid")
	}

	nd, err := a.peer.Get(ctx, c)
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

	zerolog.Ctx(ctx).Info().Str("cid", c.String()).Int64("bytes", n).Msg("Got file from peers")
	return nil
}

func (a *LabApp) connect(ctx context.Context, addrs []string) error {
	infos, err := parseAddrs(addrs)
	if err != nil {
		return err
	}

	err = a.peer.Connect(ctx, infos)
	if err != nil {
		return err
	}

	zerolog.Ctx(ctx).Info().Int("peers", len(addrs)).Msg("Connected to peers")
	return nil
}

func (a *LabApp) disconnect(ctx context.Context, addrs []string) error {
	infos, err := parseAddrs(addrs)
	if err != nil {
		return err
	}

	err = a.peer.Disconnect(ctx, infos)
	if err != nil {
		return err
	}

	zerolog.Ctx(ctx).Info().Int("peers", len(addrs)).Msg("Disconnected from peers")
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
