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

package scenariorouter

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/daemon"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/pkg/stringutil"
	"github.com/Netflix/p2plab/query"
	"github.com/rs/zerolog"
)

type router struct {
	db metadata.DB
}

func New(db metadata.DB) daemon.Router {
	return &router{db}
}

func (s *router) Routes() []daemon.Route {
	return []daemon.Route{
		// GET
		daemon.NewGetRoute("/scenarios/json", s.getScenarios),
		daemon.NewGetRoute("/scenarios/{name}/json", s.getScenarioByName),
		// POST
		daemon.NewPostRoute("/scenarios/create", s.postScenariosCreate),
		// PUT
		daemon.NewPutRoute("/scenarios/label", s.putScenariosLabel),
		// DELETE
		daemon.NewDeleteRoute("/scenarios/delete", s.deleteScenarios),
	}
}

func (s *router) getScenarios(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	scenarios, err := s.db.ListScenarios(ctx)
	if err != nil {
		return err
	}

	return daemon.WriteJSON(w, &scenarios)
}

func (s *router) getScenarioByName(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	id := vars["name"]
	scenario, err := s.db.GetScenario(ctx, id)
	if err != nil {
		return err
	}

	return daemon.WriteJSON(w, &scenario)
}

func (s *router) postScenariosCreate(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	var sdef metadata.ScenarioDefinition
	err := json.NewDecoder(r.Body).Decode(&sdef)
	if err != nil {
		return err
	}

	name := r.FormValue("name")
	zerolog.Ctx(ctx).Info().Str("scenario", name).Msg("Creating scenario")
	scenario, err := s.db.CreateScenario(ctx, metadata.Scenario{ID: name, Definition: sdef})
	if err != nil {
		return err
	}

	return daemon.WriteJSON(w, &scenario)
}

func (s *router) putScenariosLabel(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	names := strings.Split(r.FormValue("names"), ",")
	addLabels := stringutil.Coalesce(strings.Split(r.FormValue("adds"), ","))
	removeLabels := stringutil.Coalesce(strings.Split(r.FormValue("removes"), ","))

	var scenarios []metadata.Scenario
	if len(addLabels) > 0 || len(removeLabels) > 0 {
		var err error
		scenarios, err = s.db.LabelScenarios(ctx, names, addLabels, removeLabels)
		if err != nil {
			return err
		}
	}

	return daemon.WriteJSON(w, &scenarios)
}

func (s *router) deleteScenarios(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	ids := strings.Split(r.FormValue("names"), ",")

	zerolog.Ctx(ctx).Info().Strs("scenarios", ids).Msg("Deleting scenarios")
	err := s.db.DeleteScenarios(ctx, ids...)
	if err != nil {
		return err
	}

	return nil
}

func (s *router) matchScenarios(ctx context.Context, q string) ([]metadata.Scenario, error) {
	ss, err := s.db.ListScenarios(ctx)
	if err != nil {
		return nil, err
	}

	var ls []p2plab.Labeled
	for _, s := range ss {
		ls = append(ls, query.NewLabeled(s.ID, s.Labels))
	}

	mset, err := query.Execute(ctx, ls, q)
	if err != nil {
		return nil, err
	}

	var matchedScenarios []metadata.Scenario
	for _, s := range ss {
		if mset.Contains(s.ID) {
			matchedScenarios = append(matchedScenarios, s)
		}
	}

	return matchedScenarios, nil
}
