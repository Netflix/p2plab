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
	"net/http"
	"time"

	"github.com/Netflix/p2plab/metadata"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

type Labd struct {
	root   string
	addr   string
	db     *metadata.DB
	router *mux.Router
}

func New(root string) (*Labd, error) {
	db, err := metadata.NewDB(root)
	if err != nil {
		return nil, err
	}

	r := mux.NewRouter().UseEncodedPath().StrictSlash(true)
	d := &Labd{
		addr:   ":7001",
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
	clusters.HandleFunc("", d.clustersHandler).Methods("GET", "POST")
	clusters.HandleFunc("/{id}", d.clusterHandler).Methods("HEAD", "GET", "PUT", "DELETE")
	clusters.HandleFunc("/{id}/query", d.queryClusterHandler).Methods("POST")

	nodes := api.PathPrefix("/nodes").Subrouter()
	nodes.HandleFunc("", d.nodesHandler).Methods("PUT")
	nodes.HandleFunc("/{id}", d.nodeHandler).Methods("HEAD", "GET")

	scenarios := api.PathPrefix("/scenarios").Subrouter()
	scenarios.HandleFunc("", d.scenariosHandler).Methods("GET", "POST")
	scenarios.HandleFunc("/{id}", d.scenarioHandler).Methods("HEAD", "GET", "DELETE")

	benchmarks := api.PathPrefix("/benchmarks").Subrouter()
	benchmarks.HandleFunc("", d.benchmarksHandler).Methods("GET", "POST")
	benchmarks.HandleFunc("/{id}", d.benchmarkHandler).Methods("HEAD", "GET")
	benchmarks.HandleFunc("/{id}/cancel", d.cancelBenchmarkHandler).Methods("PUT")
	benchmarks.HandleFunc("/{id}/report", d.reportBenchmarkHandler).Methods("GET")
	benchmarks.HandleFunc("/{id}/logs", d.logsBenchmarkHandler).Methods("GET")
}

func (d *Labd) clustersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		d.listClusterHandler(w, r)
	case "POST":
		d.createClusterHandler(w, r)
	}
}

func (d *Labd) clusterHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "HEAD":
		d.statClusterHandler(w, r)
	case "GET":
		d.getClusterHandler(w, r)
	case "PUT":
		d.updateClusterHandler(w, r)
	case "DELETE":
		d.deleteClusterHandler(w, r)
	}
}

func (d *Labd) listClusterHandler(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("cluster/list")
}

func (d *Labd) createClusterHandler(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("cluster/create")

	cluster, err := d.db.CreateCluster(r.Context(), metadata.Cluster{ID: "hello"})
	if err != nil {
		panic(err)
	}
	log.Info().Msgf("cluster %q: %s", cluster.ID, cluster.CreatedAt)
}

func (d *Labd) statClusterHandler(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("cluster/stat")

	cluster, err := d.db.GetCluster(r.Context(), "hello")
	if err != nil {
		panic(err)
	}
	log.Info().Msgf("cluster %q: %s", cluster.ID, cluster.CreatedAt)
}

func (d *Labd) getClusterHandler(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("cluster/get")
}

func (d *Labd) updateClusterHandler(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("cluster/update")
}

func (d *Labd) deleteClusterHandler(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("cluster/delete")
}

func (d *Labd) queryClusterHandler(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("cluster/query")
}

func (d *Labd) nodesHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "PUT":
		d.labelNodeHandler(w, r)
	}
}

func (d *Labd) nodeHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "HEAD":
		d.statNodeHandler(w, r)
	case "GET":
		d.getNodeHandler(w, r)
	}
}

func (d *Labd) labelNodeHandler(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("node/label")
}

func (d *Labd) statNodeHandler(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("node/stat")
}

func (d *Labd) getNodeHandler(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("node/get")
}

func (d *Labd) scenariosHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		d.listScenarioHandler(w, r)
	case "POST":
		d.createScenarioHandler(w, r)
	}
}

func (d *Labd) scenarioHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "HEAD":
		d.statScenarioHandler(w, r)
	case "GET":
		d.getScenarioHandler(w, r)
	case "DELETE":
		d.deleteScenarioHandler(w, r)
	}
}

func (d *Labd) listScenarioHandler(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("scenario/list")
}

func (d *Labd) createScenarioHandler(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("scenario/create")
}

func (d *Labd) statScenarioHandler(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("scenario/stat")
}

func (d *Labd) getScenarioHandler(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("scenario/get")
}

func (d *Labd) deleteScenarioHandler(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("scenario/delete")
}

func (d *Labd) benchmarksHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		d.listBenchmarkHandler(w, r)
	case "POST":
		d.createBenchmarkHandler(w, r)
	}
}

func (d *Labd) benchmarkHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "HEAD":
		d.statBenchmarkHandler(w, r)
	case "GET":
		d.getBenchmarkHandler(w, r)
	}
}

func (d *Labd) listBenchmarkHandler(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("benchmark/list")
}

func (d *Labd) createBenchmarkHandler(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("benchmark/create")
}

func (d *Labd) statBenchmarkHandler(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("benchmark/stat")
}

func (d *Labd) getBenchmarkHandler(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("benchmark/get")
}

func (d *Labd) cancelBenchmarkHandler(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("benchmark/cancel")
}

func (d *Labd) reportBenchmarkHandler(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("benchmark/report")
}

func (d *Labd) logsBenchmarkHandler(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("benchmark/logs")
}
