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

package clusterrouter

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/daemon"
	"github.com/Netflix/p2plab/labd/controlapi"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/nodes"
	"github.com/Netflix/p2plab/pkg/httputil"
	"github.com/Netflix/p2plab/pkg/logutil"
	"github.com/Netflix/p2plab/pkg/stringutil"
	"github.com/Netflix/p2plab/query"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	bolt "go.etcd.io/bbolt"
)

type router struct {
	db       metadata.DB
	provider p2plab.NodeProvider
	client   *httputil.Client
}

func New(db metadata.DB, provider p2plab.NodeProvider, client *httputil.Client) daemon.Router {
	return &router{db, provider, client}
}

func (s *router) Routes() []daemon.Route {
	return []daemon.Route{
		// GET
		daemon.NewGetRoute("/clusters/json", s.getClusters),
		daemon.NewGetRoute("/clusters/{name}/json", s.getCluster),
		// POST
		daemon.NewPostRoute("/clusters/create", s.postClustersCreate),
		// PUT
		daemon.NewPutRoute("/clusters/label", s.putClustersLabel),
		daemon.NewPutRoute("/clusters/{name}/update", s.putClusterUpdate),
		// DELETE
		daemon.NewDeleteRoute("/clusters/delete", s.deleteClusters),
	}
}

func (s *router) getClusters(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	matchedClusters, err := s.matchClusters(ctx, r.FormValue("query"))
	if err != nil {
		return err
	}

	return daemon.WriteJSON(w, &matchedClusters)
}

func (s *router) getCluster(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	id := vars["name"]
	cluster, err := s.db.GetCluster(ctx, id)
	if err != nil {
		return err
	}

	return daemon.WriteJSON(w, &cluster)
}

func (s *router) postClustersCreate(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	var cdef metadata.ClusterDefinition
	err := json.NewDecoder(r.Body).Decode(&cdef)
	if err != nil {
		return err
	}

	name := r.FormValue("name")
	ctx, logger := logutil.WithResponseLogger(ctx, w)
	logger.UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.Str("name", name)
	})

	cluster, err := s.db.CreateCluster(ctx, metadata.Cluster{
		ID:         name,
		Status:     metadata.ClusterCreating,
		Definition: cdef,
	})
	if err != nil {
		return err
	}
	w.Header().Add(controlapi.ResourceID, name)

	zerolog.Ctx(ctx).Info().Msg("Creating node group")
	ng, err := s.provider.CreateNodeGroup(ctx, name, cdef)
	if err != nil {
		return err
	}

	zerolog.Ctx(ctx).Info().Msg("Updating metadata with new nodes")
	var mns []metadata.Node
	cluster.Status = metadata.ClusterConnecting
	err = s.db.Update(ctx, func(tx *bolt.Tx) error {
		var err error
		tctx := metadata.WithTransactionContext(ctx, tx)
		cluster, err = s.db.UpdateCluster(tctx, cluster)
		if err != nil {
			return err
		}

		mns, err = s.db.CreateNodes(tctx, cluster.ID, ng.Nodes)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	var ns []p2plab.Node
	for _, n := range mns {
		ns = append(ns, controlapi.NewNode(s.client, n))
	}

	err = nodes.WaitHealthy(ctx, ns)
	if err != nil {
		return err
	}

	err = nodes.Connect(ctx, ns)
	if err != nil {
		return err
	}

	zerolog.Ctx(ctx).Info().Msg("Updating cluster metadata")
	cluster.Status = metadata.ClusterCreated
	_, err = s.db.UpdateCluster(ctx, cluster)
	if err != nil {
		return err
	}

	return nil
}

func (s *router) putClustersLabel(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	names := strings.Split(r.FormValue("names"), ",")
	addLabels := stringutil.Coalesce(strings.Split(r.FormValue("adds"), ","))
	removeLabels := stringutil.Coalesce(strings.Split(r.FormValue("removes"), ","))

	var clusters []metadata.Cluster
	if len(addLabels) > 0 || len(removeLabels) > 0 {
		var err error
		clusters, err = s.db.LabelClusters(ctx, names, addLabels, removeLabels)
		if err != nil {
			return err
		}
	}

	return daemon.WriteJSON(w, &clusters)
}

func (s *router) putClusterUpdate(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	ctx, _ = logutil.WithResponseLogger(ctx, w)

	id := vars["name"]
	cluster, err := s.db.GetCluster(ctx, id)
	if err != nil {
		return err
	}

	mns, err := s.db.ListNodes(ctx, cluster.ID)
	if err != nil {
		return err
	}

	var ns []p2plab.Node
	for _, n := range mns {
		node := controlapi.NewNode(s.client, n)
		ns = append(ns, node)
	}

	url := r.FormValue("url")
	err = nodes.Update(ctx, ns, url)
	if err != nil {
		return err
	}

	return nil
}

func (s *router) deleteClusters(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	names := strings.Split(r.FormValue("names"), ",")

	ctx, logger := logutil.WithResponseLogger(ctx, w)

	// TODO: parallelize with different color loggers?
	for _, name := range names {
		logger := logger.With().Str("name", name).Logger()
		ctx = logger.WithContext(ctx)

		cluster, err := s.db.GetCluster(ctx, name)
		if err != nil {
			return errors.Wrapf(err, "failed to get cluster %q", name)
		}

		if cluster.Status != metadata.ClusterDestroying {
			cluster.Status = metadata.ClusterDestroying
			cluster, err = s.db.UpdateCluster(ctx, cluster)
			if err != nil {
				return errors.Wrap(err, "failed to update cluster status to destroying")
			}
		}

		ns, err := s.db.ListNodes(ctx, cluster.ID)
		if err != nil {
			return errors.Wrap(err, "failed to list nodes")
		}

		ng := &p2plab.NodeGroup{
			ID:    cluster.ID,
			Nodes: ns,
		}

		logger.Info().Msg("Destroying node group")
		err = s.provider.DestroyNodeGroup(ctx, ng)
		if err != nil {
			return errors.Wrap(err, "failed to destroy node group")
		}

		logger.Info().Msg("Deleting cluster metadata")
		err = s.db.DeleteCluster(ctx, cluster.ID)
		if err != nil {
			return errors.Wrap(err, "failed to delete cluster metadata")
		}

		logger.Info().Msg("Destroyed cluster")
	}

	return nil
}

func (s *router) matchClusters(ctx context.Context, q string) ([]metadata.Cluster, error) {
	cs, err := s.db.ListClusters(ctx)
	if err != nil {
		return nil, err
	}

	var ls []p2plab.Labeled
	for _, c := range cs {
		ls = append(ls, query.NewLabeled(c.ID, c.Labels))
	}

	mset, err := query.Execute(ctx, ls, q)
	if err != nil {
		return nil, err
	}

	var matchedClusters []metadata.Cluster
	for _, c := range cs {
		if mset.Contains(c.ID) {
			matchedClusters = append(matchedClusters, c)
		}
	}

	return matchedClusters, nil
}
