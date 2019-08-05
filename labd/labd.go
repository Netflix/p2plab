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

package labd

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/nodes"
	"github.com/Netflix/p2plab/pkg/httputil"
	"github.com/Netflix/p2plab/query"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

type Labd struct {
	root   string
	addr   string
	db     *metadata.DB
	router *mux.Router
}

func New(root, addr string) (*Labd, error) {
	db, err := metadata.NewDB(root)
	if err != nil {
		return nil, err
	}

	r := mux.NewRouter().UseEncodedPath().StrictSlash(true)
	d := &Labd{
		addr:   addr,
		db:     db,
		router: r,
	}
	d.registerRoutes(r)

	return d, nil
}

func (d *Labd) Serve(ctx context.Context) error {
	log.Info().Msgf("APIserver listening on %s", d.addr)
	s := &http.Server{
		Handler:      d.router,
		Addr:         d.addr,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	return s.ListenAndServe()
}

func (d *Labd) registerRoutes(r *mux.Router) {
	api := r.PathPrefix("/api/v0").Subrouter()

	clusters := api.PathPrefix("/clusters").Subrouter()
	clusters.Handle("", httputil.ErrorHandler{d.clustersHandler}).Methods("GET", "POST")
	clusters.Handle("/{cluster}", httputil.ErrorHandler{d.clusterHandler}).Methods("GET", "PUT", "DELETE")
	clusters.Handle("/{cluster}/query", httputil.ErrorHandler{d.queryClusterHandler}).Methods("POST")

	nodes := clusters.PathPrefix("/{cluster}/nodes").Subrouter()
	nodes.Handle("/{node}", httputil.ErrorHandler{d.getNodeHandler}).Methods("GET")

	scenarios := api.PathPrefix("/scenarios").Subrouter()
	scenarios.Handle("", httputil.ErrorHandler{d.scenariosHandler}).Methods("GET", "POST")
	scenarios.Handle("/{scenario}", httputil.ErrorHandler{d.scenarioHandler}).Methods("GET", "DELETE")

	benchmarks := api.PathPrefix("/benchmarks").Subrouter()
	benchmarks.Handle("", httputil.ErrorHandler{d.benchmarksHandler}).Methods("GET", "POST")
	benchmarks.Handle("/{benchmark}", httputil.ErrorHandler{d.getBenchmarkHandler}).Methods("GET")
	benchmarks.Handle("/{benchmark}/cancel", httputil.ErrorHandler{d.cancelBenchmarkHandler}).Methods("PUT")
	benchmarks.Handle("/{benchmark}/report", httputil.ErrorHandler{d.reportBenchmarkHandler}).Methods("GET")
	benchmarks.Handle("/{benchmark}/logs", httputil.ErrorHandler{d.logsBenchmarkHandler}).Methods("GET")
}

func (d *Labd) clustersHandler(w http.ResponseWriter, r *http.Request) error {
	var err error
	switch r.Method {
	case "GET":
		err = d.listClusterHandler(w, r)
	case "POST":
		err = d.createClusterHandler(w, r)
	}
	return err
}

func (d *Labd) clusterHandler(w http.ResponseWriter, r *http.Request) error {
	var err error
	switch r.Method {
	case "GET":
		err = d.getClusterHandler(w, r)
	case "PUT":
		err = d.updateClusterHandler(w, r)
	case "DELETE":
		err = d.deleteClusterHandler(w, r)
	}
	return err
}

func (d *Labd) listClusterHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("cluster/list")

	clusters, err := d.db.ListClusters(r.Context())
	if err != nil {
		return err
	}

	return httputil.WriteJSON(w, &clusters)
}

func (d *Labd) createClusterHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("cluster/create")

	cluster, err := d.db.CreateCluster(r.Context(), metadata.Cluster{ID: r.FormValue("id")})
	if err != nil {
		return err
	}

	_, err = d.db.CreateNode(r.Context(), cluster.ID, metadata.Node{ID: "node-1", Address: "http://localhost:7002"})
	if err != nil {
		return err
	}

	return httputil.WriteJSON(w, &cluster)
}

func (d *Labd) getClusterHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("cluster/get")

	vars := mux.Vars(r)
	cluster, err := d.db.GetCluster(r.Context(), vars["cluster"])
	if err != nil {
		return err
	}

	return httputil.WriteJSON(w, &cluster)
}

func (d *Labd) updateClusterHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("cluster/update")

	vars := mux.Vars(r)
	cluster, err := d.db.UpdateCluster(r.Context(), metadata.Cluster{
		ID: vars["cluster"],
	})
	if err != nil {
		return err
	}

	return httputil.WriteJSON(w, &cluster)
}

func (d *Labd) deleteClusterHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("cluster/delete")

	vars := mux.Vars(r)
	err := d.db.DeleteCluster(r.Context(), vars["cluster"])
	if err != nil {
		return err
	}

	return nil
}

func (d *Labd) queryClusterHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("cluster/query")

	q, err := query.Parse(r.FormValue("query"))
	if err != nil {
		return err
	}

	vars := mux.Vars(r)
	ns, err := d.db.ListNodes(r.Context(), vars["cluster"])
	if err != nil {
		return err
	}

	nset := nodes.NewSet()
	for _, n := range ns {
		nset.Add(&node{metadata: n})
	}

	mset, err := q.Match(r.Context(), nset)
	if err != nil {
		return err
	}

	var matchedNodes []metadata.Node
	for _, n := range ns {
		if mset.Contains(&node{metadata: n}) {
			matchedNodes = append(matchedNodes, n)
		}
	}

	addLabels := removeEmpty(strings.Split(r.FormValue("add"), ","))
	removeLabels := removeEmpty(strings.Split(r.FormValue("remove"), ","))

	if len(addLabels) > 0 || len(removeLabels) > 0 {
		var ids []string
		for _, n := range matchedNodes {
			ids = append(ids, n.ID)
		}

		matchedNodes, err = d.db.LabelNodes(r.Context(), vars["cluster"], ids, addLabels, removeLabels)
		if err != nil {
			return err
		}
	}

	return httputil.WriteJSON(w, &matchedNodes)
}

func (d *Labd) getNodeHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("node/get")

	vars := mux.Vars(r)
	node, err := d.db.GetNode(r.Context(), vars["cluster"], vars["node"])
	if err != nil {
		return err
	}

	return httputil.WriteJSON(w, &node)
}

func (d *Labd) scenariosHandler(w http.ResponseWriter, r *http.Request) error {
	var err error
	switch r.Method {
	case "GET":
		err = d.listScenarioHandler(w, r)
	case "POST":
		err = d.createScenarioHandler(w, r)
	}
	return err
}

func (d *Labd) scenarioHandler(w http.ResponseWriter, r *http.Request) error {
	var err error
	switch r.Method {
	case "GET":
		err = d.getScenarioHandler(w, r)
	case "DELETE":
		err = d.deleteScenarioHandler(w, r)
	}
	return err
}

func (d *Labd) listScenarioHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("scenario/list")

	scenarios, err := d.db.ListScenarios(r.Context())
	if err != nil {
		return err
	}

	return httputil.WriteJSON(w, &scenarios)
}

func (d *Labd) createScenarioHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("scenario/create")

	var sdef metadata.ScenarioDefinition
	err := json.NewDecoder(r.Body).Decode(&sdef)
	if err != nil {
		return err
	}

	scenario, err := d.db.CreateScenario(r.Context(), metadata.Scenario{ID: r.FormValue("id"), Definition: sdef})
	if err != nil {
		return err
	}

	return httputil.WriteJSON(w, &scenario)
}

func (d *Labd) getScenarioHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("scenario/get")

	vars := mux.Vars(r)
	scenario, err := d.db.GetScenario(r.Context(), vars["scenario"])
	if err != nil {
		return err
	}

	return httputil.WriteJSON(w, &scenario)
}

func (d *Labd) deleteScenarioHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("scenario/delete")

	vars := mux.Vars(r)
	err := d.db.DeleteScenario(r.Context(), vars["scenario"])
	if err != nil {
		return err
	}

	return nil
}

func (d *Labd) benchmarksHandler(w http.ResponseWriter, r *http.Request) error {
	var err error
	switch r.Method {
	case "GET":
		err = d.listBenchmarkHandler(w, r)
	case "POST":
		err = d.createBenchmarkHandler(w, r)
	}
	return err
}

func (d *Labd) listBenchmarkHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("benchmark/list")

	benchmarks, err := d.db.ListBenchmarks(r.Context())
	if err != nil {
		return err
	}

	return httputil.WriteJSON(w, &benchmarks)
}

func (d *Labd) createBenchmarkHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("benchmark/create")

	ctx := r.Context()
	sid := r.FormValue("scenario")
	scenario, err := d.db.GetScenario(ctx, sid)
	if err != nil {
		return err
	}

	cid := r.FormValue("cluster")
	cluster, err := d.db.GetCluster(ctx, cid)
	if err != nil {
		return err
	}

	benchmark := metadata.Benchmark{
		ID:       time.Now().Format(time.RFC3339Nano),
		Cluster:  cluster,
		Scenario: scenario,
	}

	benchmark, err = d.db.CreateBenchmark(ctx, benchmark)
	if err != nil {
		return err
	}

	return httputil.WriteJSON(w, &benchmark)
}

func (d *Labd) getBenchmarkHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("benchmark/get")

	vars := mux.Vars(r)
	benchmark, err := d.db.GetBenchmark(r.Context(), vars["benchmark"])
	if err != nil {
		return err
	}

	return httputil.WriteJSON(w, &benchmark)
}

func (d *Labd) cancelBenchmarkHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("benchmark/cancel")

	vars := mux.Vars(r)
	benchmark, err := d.db.GetBenchmark(r.Context(), vars["benchmark"])
	if err != nil {
		return err
	}
	log.Info().Msgf("cancel %q", benchmark.ID)

	return nil
}

func (d *Labd) reportBenchmarkHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("benchmark/report")

	vars := mux.Vars(r)
	benchmark, err := d.db.GetBenchmark(r.Context(), vars["benchmark"])
	if err != nil {
		return err
	}
	log.Info().Msgf("report %q", benchmark.ID)

	return nil
}

func (d *Labd) logsBenchmarkHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("benchmark/logs")

	vars := mux.Vars(r)
	benchmark, err := d.db.GetBenchmark(r.Context(), vars["benchmark"])
	if err != nil {
		return err
	}
	log.Info().Msgf("logs %q", benchmark.ID)

	return nil
}

func removeEmpty(slice []string) []string {
	var r []string
	for _, e := range slice {
		if e == "" {
			continue
		}
		r = append(r, e)
	}
	return r
}
