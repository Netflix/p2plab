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

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/metadata"
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
	clusters.Handle("", ErrorHandler{d.clustersHandler}).Methods("GET", "POST")
	clusters.Handle("/{id}", ErrorHandler{d.clusterHandler}).Methods("GET", "PUT", "DELETE")
	clusters.Handle("/{id}/query", ErrorHandler{d.queryClusterHandler}).Methods("POST")

	nodes := api.PathPrefix("/nodes").Subrouter()
	nodes.Handle("/{id}", ErrorHandler{d.nodeHandler}).Methods("GET")

	scenarios := api.PathPrefix("/scenarios").Subrouter()
	scenarios.Handle("", ErrorHandler{d.scenariosHandler}).Methods("GET", "POST")
	scenarios.Handle("/{id}", ErrorHandler{d.scenarioHandler}).Methods("GET", "DELETE")

	benchmarks := api.PathPrefix("/benchmarks").Subrouter()
	benchmarks.Handle("", ErrorHandler{d.benchmarksHandler}).Methods("GET", "POST")
	benchmarks.Handle("/{id}", ErrorHandler{d.benchmarkHandler}).Methods("GET")
	benchmarks.Handle("/{id}/cancel", ErrorHandler{d.cancelBenchmarkHandler}).Methods("PUT")
	benchmarks.Handle("/{id}/report", ErrorHandler{d.reportBenchmarkHandler}).Methods("GET")
	benchmarks.Handle("/{id}/logs", ErrorHandler{d.logsBenchmarkHandler}).Methods("GET")
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

	return writeJSON(w, &clusters)
}

func (d *Labd) createClusterHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("cluster/create")

	cluster, err := d.db.CreateCluster(r.Context(), metadata.Cluster{ID: r.FormValue("id")})
	if err != nil {
		return err
	}

	// _, err = d.db.CreateNode(r.Context(), cluster.ID, metadata.Node{ID: "node-1"})
	// if err != nil {
	// 	return err
	// }

	return writeJSON(w, &cluster)
}

func (d *Labd) getClusterHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("cluster/get")

	vars := mux.Vars(r)
	cluster, err := d.db.GetCluster(r.Context(), vars["id"])
	if err != nil {
		return err
	}

	return writeJSON(w, &cluster)
}

func (d *Labd) updateClusterHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("cluster/update")

	vars := mux.Vars(r)
	cluster, err := d.db.UpdateCluster(r.Context(), metadata.Cluster{
		ID: vars["id"],
	})
	if err != nil {
		return err
	}

	return writeJSON(w, &cluster)
}

func (d *Labd) deleteClusterHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("cluster/delete")

	vars := mux.Vars(r)
	err := d.db.DeleteCluster(r.Context(), vars["id"])
	if err != nil {
		return err
	}

	return nil
}

func (d *Labd) queryClusterHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("cluster/query")

	vars := mux.Vars(r)
	q, err := query.Parse(vars["query"])
	if err != nil {
		return err
	}

	nodes, err := d.db.ListNodes(r.Context(), vars["id"])
	if err != nil {
		return err
	}

	nset := nodeSet{
		set: make(map[string]p2plab.Node),
	}
	for _, n := range nodes {
		nset.Add(&node{metadata: n})
	}

	mset, err := q.Match(r.Context(), &nset)
	if err != nil {
		return err
	}

	var matchedNodes []metadata.Node
	for _, n := range nodes {
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

		matchedNodes, err = d.db.LabelNodes(r.Context(), vars["id"], ids, addLabels, removeLabels)
		if err != nil {
			return err
		}
	}

	return writeJSON(w, &matchedNodes)
}

func (d *Labd) nodesHandler(w http.ResponseWriter, r *http.Request) error {
	var err error
	switch r.Method {
	case "PUT":
		err = d.labelNodeHandler(w, r)
	}
	return err
}

func (d *Labd) nodeHandler(w http.ResponseWriter, r *http.Request) error {
	var err error
	switch r.Method {
	case "GET":
		err = d.getNodeHandler(w, r)
	}
	return err
}

func (d *Labd) labelNodeHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("node/label")

	return nil
}

func (d *Labd) getNodeHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("node/get")
	return nil
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
	return nil
}

func (d *Labd) createScenarioHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("scenario/create")
	return nil
}

func (d *Labd) getScenarioHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("scenario/get")
	return nil
}

func (d *Labd) deleteScenarioHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("scenario/delete")
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

func (d *Labd) benchmarkHandler(w http.ResponseWriter, r *http.Request) error {
	var err error
	switch r.Method {
	case "GET":
		err = d.getBenchmarkHandler(w, r)
	}
	return err
}

func (d *Labd) listBenchmarkHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("benchmark/list")
	return nil
}

func (d *Labd) createBenchmarkHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("benchmark/create")
	return nil
}

func (d *Labd) getBenchmarkHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("benchmark/get")
	return nil
}

func (d *Labd) cancelBenchmarkHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("benchmark/cancel")
	return nil
}

func (d *Labd) reportBenchmarkHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("benchmark/report")
	return nil
}

func (d *Labd) logsBenchmarkHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("benchmark/logs")
	return nil
}

func writeJSON(w http.ResponseWriter, v interface{}) error {
	content, err := json.MarshalIndent(v, "", "    ")
	if err != nil {
		return err
	}
	w.Write(content)
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
