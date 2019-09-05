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

package noderouter

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/daemon"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/pkg/httputil"
	"github.com/Netflix/p2plab/pkg/stringutil"
	"github.com/Netflix/p2plab/query"
	bolt "go.etcd.io/bbolt"
)

type router struct {
	db     metadata.DB
	client *httputil.Client
}

func New(db metadata.DB, client *httputil.Client) daemon.Router {
	return &router{db, client}
}

func (s *router) Routes() []daemon.Route {
	return []daemon.Route{
		// GET
		daemon.NewGetRoute("/clusters/{name}/nodes/json", s.getNodes),
		daemon.NewGetRoute("/clusters/{name}/nodes/{id}/json", s.getNodeById),
		// PUT
		daemon.NewPutRoute("/clusters/{name}/nodes/label", s.putNodesLabel),
		daemon.NewPutRoute("/clusters/{name}/nodes/update", s.putNodesUpdate),
	}
}

func (s *router) getNodes(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	clusterId := vars["name"]
	matchedNodes, err := s.matchNodes(ctx, clusterId, r.FormValue("query"))
	if err != nil {
		return err
	}

	return daemon.WriteJSON(w, &matchedNodes)
}

func (s *router) getNodeById(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	clusterId, id := vars["name"], vars["id"]
	node, err := s.db.GetNode(ctx, clusterId, id)
	if err != nil {
		return err
	}

	return daemon.WriteJSON(w, &node)
}

func (s *router) putNodesLabel(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	ids := strings.Split(r.FormValue("ids"), ",")
	addLabels := stringutil.Coalesce(strings.Split(r.FormValue("adds"), ","))
	removeLabels := stringutil.Coalesce(strings.Split(r.FormValue("removes"), ","))

	var nodes []metadata.Node
	if len(addLabels) > 0 || len(removeLabels) > 0 {
		var err error
		clusterId := vars["name"]
		nodes, err = s.db.LabelNodes(ctx, clusterId, ids, addLabels, removeLabels)
		if err != nil {
			return err
		}
	}

	return daemon.WriteJSON(w, &nodes)
}

func (s *router) putNodesUpdate(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	var pdef metadata.PeerDefinition
	err := json.NewDecoder(r.Body).Decode(&pdef)
	if err != nil {
		return err
	}

	clusterId := vars["name"]
	matchedNodes, err := s.matchNodes(ctx, clusterId, r.FormValue("query"))
	if err != nil {
		return err
	}

	var ns []metadata.Node
	err = s.db.Update(ctx, func(tx *bolt.Tx) error {
		tctx := metadata.WithTransactionContext(ctx, tx)

		for _, n := range matchedNodes {
			if pdef.GitReference != "" {
				n.Peer.GitReference = pdef.GitReference
			}
			if len(pdef.Transports) > 0 {
				n.Peer.Transports = pdef.Transports
			}
			if len(pdef.Muxers) > 0 {
				n.Peer.Muxers = pdef.Muxers
			}
			if len(pdef.SecurityTransports) > 0 {
				n.Peer.SecurityTransports = pdef.SecurityTransports
			}
			if pdef.Routing != "" {
				n.Peer.Routing = pdef.Routing
			}

			var err error
			n, err = s.db.UpdateNode(tctx, clusterId, n)
			if err != nil {
				return err
			}
			ns = append(ns, n)
		}

		return nil
	})
	if err != nil {
		return err
	}

	return daemon.WriteJSON(w, &ns)
}

func (s *router) matchNodes(ctx context.Context, clusterId, q string) ([]metadata.Node, error) {
	ns, err := s.db.ListNodes(ctx, clusterId)
	if err != nil {
		return nil, err
	}

	var ls []p2plab.Labeled
	for _, n := range ns {
		ls = append(ls, query.NewLabeled(n.ID, n.Labels))
	}

	mset, err := query.Execute(ctx, ls, q)
	if err != nil {
		return nil, err
	}

	var matchedNodes []metadata.Node
	for _, n := range ns {
		if mset.Contains(n.ID) {
			matchedNodes = append(matchedNodes, n)
		}
	}

	return matchedNodes, nil
}
