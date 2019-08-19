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
	"net/http"
	"strings"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/daemon"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/peer"
	"github.com/Netflix/p2plab/pkg/httputil"
	"github.com/Netflix/p2plab/pkg/stringutil"
	"github.com/Netflix/p2plab/query"
	"github.com/Netflix/p2plab/transformers"
	"github.com/containerd/containerd/errdefs"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type router struct {
	db       metadata.DB
	provider p2plab.NodeProvider
	client   *httputil.Client
	ts       *transformers.Transformers
	seeder   *peer.Peer
}

func New(db metadata.DB, provider p2plab.NodeProvider, client *httputil.Client, ts *transformers.Transformers, seeder *peer.Peer) daemon.Router {
	return &router{db, provider, client, ts, seeder}
}

func (s *router) Routes() []daemon.Route {
	return []daemon.Route{
		// GET
		daemon.NewGetRoute("/experiments/json", s.getExperiments),
		daemon.NewGetRoute("/experiments/{id}/json", s.getExperimentByName),
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

func (s *router) getExperimentByName(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	id := vars["name"]
	experiment, err := s.db.GetExperiment(ctx, id)
	if err != nil {
		return err
	}

	return daemon.WriteJSON(w, &experiment)
}

func (s *router) postExperimentsCreate(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	return nil
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
