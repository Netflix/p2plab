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

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

type Labd struct {
	addr   string
	router *mux.Router
}

func New() (*Labd, error) {
	d := &Labd{
		addr:   ":8080",
		router: mux.NewRouter(),
	}
	d.registerRoutes(d.router)

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
	clusters.HandleFunc("/", d.listClusterHandler).Methods("GET")
	clusters.HandleFunc("/", d.createClusterHandler).Methods("POST")
	clusters.HandleFunc("/{id}", d.statClusterHandler).Methods("HEAD")
	clusters.HandleFunc("/{id}", d.getClusterHandler).Methods("GET")
	clusters.HandleFunc("/{id}", d.updateClusterHandler).Methods("PUT")
	clusters.HandleFunc("/{id}", d.deleteClusterHandler).Methods("DELETE")
	clusters.HandleFunc("/{id}/query", d.queryClusterHandler).Methods("POST")

	nodes := api.PathPrefix("/nodes").Subrouter()
	nodes.HandleFunc("/{id}", d.statNodeHandler).Methods("HEAD")
	nodes.HandleFunc("/{id}", d.getNodeHandler).Methods("GET")

	scenarios := api.PathPrefix("/scenarios").Subrouter()
	scenarios.HandleFunc("/", d.listScenarioHandler).Methods("GET")
	scenarios.HandleFunc("/", d.createScenarioHandler).Methods("POST")
	scenarios.HandleFunc("/{id}", d.statScenarioHandler).Methods("HEAD")
	scenarios.HandleFunc("/{id}", d.getScenarioHandler).Methods("GET")
	scenarios.HandleFunc("/{id}", d.deleteScenarioHandler).Methods("DELETE")

	benchmarks := api.PathPrefix("/benchmarks").Subrouter()
	benchmarks.HandleFunc("/", d.listBenchmarkHandler).Methods("GET")
	benchmarks.HandleFunc("/", d.createBenchmarkHandler).Methods("POST")
	benchmarks.HandleFunc("/{id}", d.statBenchmarkHandler).Methods("HEAD")
	benchmarks.HandleFunc("/{id}", d.getBenchmarkHandler).Methods("GET")
	benchmarks.HandleFunc("/{id}/cancel", d.cancelBenchmarkHandler).Methods("PUT")
	benchmarks.HandleFunc("/{id}/report", d.reportBenchmarkHandler).Methods("GET")
	benchmarks.HandleFunc("/{id}/logs", d.logsBenchmarkHandler).Methods("GET")
}

func (d *Labd) listClusterHandler(http.ResponseWriter, *http.Request) {
}

func (d *Labd) createClusterHandler(http.ResponseWriter, *http.Request) {
}

func (d *Labd) statClusterHandler(http.ResponseWriter, *http.Request) {
}

func (d *Labd) getClusterHandler(http.ResponseWriter, *http.Request) {
}

func (d *Labd) updateClusterHandler(http.ResponseWriter, *http.Request) {
}

func (d *Labd) deleteClusterHandler(http.ResponseWriter, *http.Request) {
}

func (d *Labd) queryClusterHandler(http.ResponseWriter, *http.Request) {
}

func (d *Labd) statNodeHandler(http.ResponseWriter, *http.Request) {
}

func (d *Labd) getNodeHandler(http.ResponseWriter, *http.Request) {
}

func (d *Labd) listScenarioHandler(http.ResponseWriter, *http.Request) {
}

func (d *Labd) createScenarioHandler(http.ResponseWriter, *http.Request) {
}

func (d *Labd) statScenarioHandler(http.ResponseWriter, *http.Request) {
}

func (d *Labd) getScenarioHandler(http.ResponseWriter, *http.Request) {
}

func (d *Labd) deleteScenarioHandler(http.ResponseWriter, *http.Request) {
}

func (d *Labd) listBenchmarkHandler(http.ResponseWriter, *http.Request) {
}

func (d *Labd) createBenchmarkHandler(http.ResponseWriter, *http.Request) {
}

func (d *Labd) statBenchmarkHandler(http.ResponseWriter, *http.Request) {
}

func (d *Labd) getBenchmarkHandler(http.ResponseWriter, *http.Request) {
}

func (d *Labd) cancelBenchmarkHandler(http.ResponseWriter, *http.Request) {
}

func (d *Labd) reportBenchmarkHandler(http.ResponseWriter, *http.Request) {
}

func (d *Labd) logsBenchmarkHandler(http.ResponseWriter, *http.Request) {
}
