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
	"net/http"
	"time"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/errdefs"
	"github.com/Netflix/p2plab/peer"
	"github.com/Netflix/p2plab/pkg/httputil"
	"github.com/gorilla/mux"
	cid "github.com/ipfs/go-cid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type LabApp struct {
	root   string
	addr   string
	router *mux.Router
	peer   p2plab.Peer
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
	a.peer, err = peer.NewPeer(ctx, a.root)
	if err != nil {
		return errors.Wrap(err, "failed to create peer peer")
	}

	var addrs []string
	for _, ma := range a.peer.Host().Addrs() {
		addrs = append(addrs, ma.String())
	}
	log.Info().Msgf("IPFS listening on %s", addrs)

	s := &http.Server{
		Handler:      a.router,
		Addr:         a.addr,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	log.Info().Msgf("labapp listening on %s", a.addr)

	return s.ListenAndServe()
}

func (a *LabApp) registerRoutes(r *mux.Router) {
	api := r.PathPrefix("/api/v0").Subrouter()
	api.Handle("/run", httputil.ErrorHandler{a.runHandler}).Methods("POST")
}

func (a *LabApp) runHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("labapp/run")

	var task p2plab.Task
	err := json.NewDecoder(r.Body).Decode(&task)
	if err != nil {
		return err
	}

	switch task.Type {
	case p2plab.TaskGetDAG:
		err = a.getFile(r.Context(), task.Target)
	default:
		return errors.Wrapf(errdefs.ErrInvalidArgument, "unrecognized task type: %q", task.Type)
	}

	resp := TaskResponse{
		Err: err.Error(),
	}
	return httputil.WriteJSON(w, &resp)
}

func (a *LabApp) getFile(ctx context.Context, target string) error {
	targetCid, err := cid.Parse(target)
	if err != nil {
		return err
	}

	r, err := a.peer.Get(ctx, targetCid)
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
