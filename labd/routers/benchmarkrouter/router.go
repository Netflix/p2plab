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

package benchmarkrouter

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/daemon"
	"github.com/Netflix/p2plab/labd/controlapi"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/nodes"
	"github.com/Netflix/p2plab/peer"
	"github.com/Netflix/p2plab/pkg/httputil"
	"github.com/Netflix/p2plab/pkg/logutil"
	"github.com/Netflix/p2plab/pkg/stringutil"
	"github.com/Netflix/p2plab/query"
	"github.com/Netflix/p2plab/scenarios"
	"github.com/Netflix/p2plab/transformers"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type router struct {
	db       metadata.DB
	client   *httputil.Client
	ts       *transformers.Transformers
	seeder   *peer.Peer
	uploader p2plab.Uploader
}

func New(db metadata.DB, client *httputil.Client, ts *transformers.Transformers, seeder *peer.Peer, uploader p2plab.Uploader) daemon.Router {
	return &router{db, client, ts, seeder, uploader}
}

func (s *router) Routes() []daemon.Route {
	return []daemon.Route{
		// GET
		daemon.NewGetRoute("/benchmarks/json", s.getBenchmarks),
		daemon.NewGetRoute("/benchmarks/{id}/json", s.getBenchmarkById),
		// POST
		daemon.NewPostRoute("/benchmarks/create", s.postBenchmarksCreate),
		// PUT
		daemon.NewPutRoute("/benchmarks/label", s.putBenchmarksLabel),
		// DELETE
		daemon.NewDeleteRoute("/benchmarks/delete", s.deleteBenchmarks),
	}
}

func (s *router) getBenchmarks(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	benchmarks, err := s.db.ListBenchmarks(ctx)
	if err != nil {
		return err
	}

	return daemon.WriteJSON(w, &benchmarks)
}

func (s *router) getBenchmarkById(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	id := vars["id"]
	benchmark, err := s.db.GetBenchmark(ctx, id)
	if err != nil {
		return err
	}

	return daemon.WriteJSON(w, &benchmark)
}

func (s *router) postBenchmarksCreate(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	noReset := false
	if r.FormValue("no-reset") != "" {
		var err error
		noReset, err = strconv.ParseBool(r.FormValue("no-reset"))
		if err != nil {
			return err
		}
	}

	sid := r.FormValue("scenario")
	scenario, err := s.db.GetScenario(ctx, sid)
	if err != nil {
		return err
	}

	cid := r.FormValue("cluster")
	cluster, err := s.db.GetCluster(ctx, cid)
	if err != nil {
		return err
	}

	bid := fmt.Sprintf("%s-%s-%d", sid, cid, time.Now().UnixNano())
	w.Header().Add(controlapi.ResourceID, bid)

	ctx, logger := logutil.WithResponseLogger(ctx, w)
	logger.UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.Str("bid", bid)
	})

	zerolog.Ctx(ctx).Info().Msg("Retrieving nodes in cluster")
	mns, err := s.db.ListNodes(ctx, cid)
	if err != nil {
		return err
	}

	var ns []p2plab.Node
	lset := query.NewLabeledSet()
	for _, n := range mns {
		node := controlapi.NewNode(s.client, n)
		lset.Add(node)
		ns = append(ns, node)
	}

	if !noReset {
		err = nodes.Update(ctx, ns, "")
		if err != nil {
			return errors.Wrap(err, "failed to update cluster")
		}

		err = nodes.Connect(ctx, ns)
		if err != nil {
			return errors.Wrap(err, "failed to connect cluster")
		}
	}

	zerolog.Ctx(ctx).Info().Msg("Creating scenario plan")
	plan, err := scenarios.Plan(ctx, scenario.Definition, s.ts, s.seeder, lset)
	if err != nil {
		return errors.Wrap(err, "failed to create scenario plan")
	}

	benchmark := metadata.Benchmark{
		ID:       bid,
		Status:   metadata.BenchmarkRunning,
		Cluster:  cluster,
		Scenario: scenario,
		Plan:     plan,
		Labels: []string{
			cid,
			sid,
		},
	}

	zerolog.Ctx(ctx).Info().Msg("Creating benchmark metadata")
	benchmark, err = s.db.CreateBenchmark(ctx, benchmark)
	if err != nil {
		return err
	}

	seederAddr := fmt.Sprintf("%s/p2p/%s", s.seeder.Host().Addrs()[1], s.seeder.Host().ID())
	zerolog.Ctx(ctx).Info().Msg("Executing scenario plan")
	err = scenarios.Run(ctx, lset, plan, seederAddr)
	if err != nil {
		return errors.Wrap(err, "failed to run scenario plan")
	}

	zerolog.Ctx(ctx).Info().Msg("Updating benchmark metadata")
	benchmark.Status = metadata.BenchmarkDone
	_, err = s.db.UpdateBenchmark(ctx, benchmark)
	if err != nil {
		return errors.Wrap(err, "failed to update benchmark")
	}

	return nil
}

func (s *router) putBenchmarksLabel(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	ids := strings.Split(r.FormValue("ids"), ",")
	addLabels := stringutil.Coalesce(strings.Split(r.FormValue("adds"), ","))
	removeLabels := stringutil.Coalesce(strings.Split(r.FormValue("removes"), ","))

	var benchmarks []metadata.Benchmark
	if len(addLabels) > 0 || len(removeLabels) > 0 {
		var err error
		benchmarks, err = s.db.LabelBenchmarks(ctx, ids, addLabels, removeLabels)
		if err != nil {
			return err
		}
	}

	return daemon.WriteJSON(w, &benchmarks)
}

func (s *router) deleteBenchmarks(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	ids := strings.Split(r.FormValue("ids"), ",")

	err := s.db.DeleteBenchmarks(ctx, ids...)
	if err != nil {
		return err
	}

	return nil
}

func (s *router) matchBenchmarks(ctx context.Context, q string) ([]metadata.Benchmark, error) {
	bs, err := s.db.ListBenchmarks(ctx)
	if err != nil {
		return nil, err
	}

	var ls []p2plab.Labeled
	for _, b := range bs {
		ls = append(ls, query.NewLabeled(b.ID, b.Labels))
	}

	mset, err := query.Execute(ctx, ls, q)
	if err != nil {
		return nil, err
	}

	var matchedBenchmarks []metadata.Benchmark
	for _, b := range bs {
		if mset.Contains(b.ID) {
			matchedBenchmarks = append(matchedBenchmarks, b)
		}
	}

	return matchedBenchmarks, nil
}
