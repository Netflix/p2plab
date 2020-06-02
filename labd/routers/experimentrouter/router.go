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

package experimentrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/daemon"
	"github.com/Netflix/p2plab/labd/controlapi"
	"github.com/Netflix/p2plab/labd/routers/helpers"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/nodes"
	"github.com/Netflix/p2plab/peer"
	"github.com/Netflix/p2plab/pkg/httputil"
	"github.com/Netflix/p2plab/pkg/logutil"
	"github.com/Netflix/p2plab/pkg/stringutil"
	"github.com/Netflix/p2plab/query"
	"github.com/Netflix/p2plab/reports"
	"github.com/Netflix/p2plab/scenarios"
	"github.com/Netflix/p2plab/transformers"
	"github.com/containerd/containerd/errdefs"
	"github.com/pkg/errors"
	"github.com/rs/xid"
	"github.com/rs/zerolog"
	jaeger "github.com/uber/jaeger-client-go"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/sync/errgroup"
)

type router struct {
	db       metadata.DB
	provider p2plab.NodeProvider
	client   *httputil.Client
	ts       *transformers.Transformers
	seeder   *peer.Peer
	builder  p2plab.Builder
	rhelper  *helpers.Helper
}

// New returns a new experiment router initialized with the router helpers
func New(db metadata.DB, provider p2plab.NodeProvider, client *httputil.Client, ts *transformers.Transformers, seeder *peer.Peer, builder p2plab.Builder) daemon.Router {
	return &router{
		db,
		provider,
		client,
		ts,
		seeder,
		builder,
		helpers.New(db, provider, client),
	}
}

func (s *router) Routes() []daemon.Route {
	return []daemon.Route{
		// GET
		daemon.NewGetRoute("/experiments/json", s.getExperiments),
		daemon.NewGetRoute("/experiments/{id}/json", s.getExperimentByID),
		// POST
		daemon.NewPostRoute("/experiments/create", s.postExperimentsCreate),
		// PUT
		daemon.NewPutRoute("/experiments/label", s.putExperimentsLabel),
		// DELETE
		daemon.NewDeleteRoute("/experiments/delete", s.deleteExperiments),
	}
}

func (s *router) getExperiments(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	experiments, err := s.db.ListExperiments(ctx)
	if err != nil {
		return err
	}

	return daemon.WriteJSON(w, &experiments)
}

func (s *router) getExperimentByID(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	id := vars["id"]
	experiment, err := s.db.GetExperiment(ctx, id)
	if err != nil {
		return err
	}

	return daemon.WriteJSON(w, &experiment)
}

func (s *router) postExperimentsCreate(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	var edef metadata.ExperimentDefinition
	err := json.NewDecoder(r.Body).Decode(&edef)
	if err != nil {
		return err
	}

	eid := xid.New().String()
	w.Header().Add(controlapi.ResourceID, eid)

	ctx, logger := logutil.WithResponseLogger(ctx, w)
	logger.UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.Str("eid", eid)
	})

	experiment, err := s.db.CreateExperiment(ctx, metadata.Experiment{
		ID:         eid,
		Definition: edef,
		Status:     metadata.ExperimentRunning,
	})
	if err != nil {
		return err
	}

	var seederAddrs []string
	for _, addr := range s.seeder.Host().Addrs() {
		seederAddrs = append(seederAddrs, fmt.Sprintf("%s/p2p/%s", addr, s.seeder.Host().ID()))
	}

	var mu sync.Mutex
	experiment.Reports = make([]metadata.Report, len(experiment.Definition.Trials))

	eg, ctx := errgroup.WithContext(ctx)
	for i, trial := range experiment.Definition.Trials {
		i, trial := i, trial
		name := fmt.Sprintf("experiment_%s_trial_%d", eid, i)

		eg.Go(func() error {
			cluster, err := s.rhelper.CreateCluster(ctx, trial.Cluster, name)
			if err != nil {
				return err
			}

			defer func() error {
				return s.rhelper.DeleteCluster(ctx, name)
			}()

			mns, err := s.db.ListNodes(ctx, cluster.ID)
			if err != nil {
				return err
			}

			var (
				ns   []p2plab.Node
				lset = query.NewLabeledSet()
			)
			for _, n := range mns {
				node := controlapi.NewNode(s.client, n)
				lset.Add(node)
				ns = append(ns, node)
			}

			var ids []string
			for _, labeled := range lset.Slice() {
				ids = append(ids, labeled.ID())
			}
			zerolog.Ctx(ctx).Info().Int("trial", i).Strs("ids", ids).Msg("Created cluster for experiment")

			err = nodes.Update(ctx, s.builder, ns)
			if err != nil {
				return errors.Wrap(err, "failed to update cluster")
			}

			err = nodes.Connect(ctx, ns)
			if err != nil {
				return errors.Wrap(err, "failed to connect cluster")
			}

			plan, queries, err := scenarios.Plan(ctx, trial.Scenario, s.ts, s.seeder, lset)
			if err != nil {
				return err
			}

			execution, err := scenarios.Run(ctx, lset, plan, seederAddrs)
			if err != nil {
				return errors.Wrapf(err, "failed to run scenario plan for %q", cluster.ID)
			}

			report := metadata.Report{
				Summary: metadata.ReportSummary{
					TotalTime: execution.End.Sub(execution.Start),
				},
				Nodes:   execution.Report,
				Queries: queries,
			}

			report.Aggregates = reports.ComputeAggregates(report.Nodes)
			jaegerUI := os.Getenv("JAEGER_UI")
			if jaegerUI != "" {
				sc, ok := execution.Span.Context().(jaeger.SpanContext)
				if ok {
					report.Summary.Trace = fmt.Sprintf("%s/trace/%s", jaegerUI, sc.TraceID())
				}
			}

			mu.Lock()
			experiment.Reports[i] = report
			mu.Unlock()
			return err
		})
	}

	err = eg.Wait()
	if err != nil {
		return err
	}

	experiment.Status = metadata.ExperimentDone
	return s.db.Update(ctx, func(tx *bolt.Tx) error {
		tctx := metadata.WithTransactionContext(ctx, tx)
		_, err := s.db.UpdateExperiment(tctx, experiment)
		if err != nil {
			return err
		}
		return nil
	})
}

func (s *router) putExperimentsLabel(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	ids := strings.Split(r.FormValue("ids"), ",")
	addLabels := stringutil.Coalesce(strings.Split(r.FormValue("adds"), ","))
	removeLabels := stringutil.Coalesce(strings.Split(r.FormValue("removes"), ","))

	var experiments []metadata.Experiment
	if len(addLabels) > 0 || len(removeLabels) > 0 {
		var err error
		experiments, err = s.db.LabelExperiments(ctx, ids, addLabels, removeLabels)
		if err != nil {
			return err
		}
	}

	return daemon.WriteJSON(w, &experiments)
}

func (s *router) deleteExperiments(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	ids := strings.Split(r.FormValue("ids"), ",")

	for _, id := range ids {
		logger := zerolog.Ctx(ctx).With().Str("experiment", id).Logger()

		experiment, err := s.db.GetExperiment(ctx, id)
		if err != nil {
			return err
		}

		switch experiment.Status {
		case metadata.ExperimentDone, metadata.ExperimentError:
		default:
			return errors.Wrapf(errdefs.ErrFailedPrecondition, "experiment status %q", experiment.Status)
		}

		logger.Info().Msg("Deleting experiment")
		err = s.db.DeleteExperiment(ctx, id)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *router) matchExperiments(ctx context.Context, q string) ([]metadata.Experiment, error) {
	es, err := s.db.ListExperiments(ctx)
	if err != nil {
		return nil, err
	}

	var ls []p2plab.Labeled
	for _, e := range es {
		ls = append(ls, query.NewLabeled(e.ID, e.Labels))
	}

	mset, err := query.Execute(ctx, ls, q)
	if err != nil {
		return nil, err
	}

	var matchedExperiments []metadata.Experiment
	for _, e := range es {
		if mset.Contains(e.ID) {
			matchedExperiments = append(matchedExperiments, e)
		}
	}

	return matchedExperiments, nil
}
