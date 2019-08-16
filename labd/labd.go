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
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/nodes"
	"github.com/Netflix/p2plab/peer"
	"github.com/Netflix/p2plab/pkg/httputil"
	"github.com/Netflix/p2plab/providers"
	"github.com/Netflix/p2plab/query"
	"github.com/Netflix/p2plab/scenarios"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	bolt "go.etcd.io/bbolt"
)

type Labd struct {
	root     string
	addr     string
	db       *metadata.DB
	router   *mux.Router
	seeder   *peer.Peer
	provider p2plab.NodeProvider
}

func New(root, addr string) (*Labd, error) {
	db, err := metadata.NewDB(root)
	if err != nil {
		return nil, err
	}

	provider, err := providers.GetNodeProvider(filepath.Join(root, "providers"), "terraform")
	if err != nil {
		return nil, err
	}

	r := mux.NewRouter().UseEncodedPath().StrictSlash(true)
	d := &Labd{
		root:     root,
		addr:     addr,
		db:       db,
		router:   r,
		provider: provider,
	}
	d.registerRoutes(r)

	return d, nil
}

func (d *Labd) Serve(ctx context.Context) error {
	var err error
	d.seeder, err = peer.New(ctx, filepath.Join(d.root, "seeder"))
	if err != nil {
		return errors.Wrap(err, "failed to create seeder peer")
	}

	var addrs []string
	for _, ma := range d.seeder.Host().Addrs() {
		addrs = append(addrs, ma.String())
	}
	log.Info().Msgf("IPFS listening on %s", addrs)

	log.Info().Msgf("labd listening on %s", d.addr)
	s := &http.Server{
		Handler:     d.router,
		Addr:        d.addr,
		ReadTimeout: 10 * time.Second,
	}
	return s.ListenAndServe()
}

func (d *Labd) registerRoutes(r *mux.Router) {
	api := r.PathPrefix("/api/v0").Subrouter()

	api.HandleFunc("/healthcheck", d.healthcheckHandler).Methods("GET")

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

	experiments := api.PathPrefix("/experiments").Subrouter()
	experiments.Handle("", httputil.ErrorHandler{d.experimentsHandler}).Methods("GET", "POST")
	experiments.Handle("/{experiment}", httputil.ErrorHandler{d.getExperimentHandler}).Methods("GET")
	experiments.Handle("/{experiment}/cancel", httputil.ErrorHandler{d.cancelExperimentHandler}).Methods("PUT")
}

func (d *Labd) healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
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

	var cdef metadata.ClusterDefinition
	err := json.NewDecoder(r.Body).Decode(&cdef)
	if err != nil {
		return err
	}

	ctx := r.Context()
	id := r.FormValue("id")
	cluster, err := d.db.CreateCluster(ctx, metadata.Cluster{
		ID:         id,
		Status:     metadata.ClusterCreating,
		Definition: cdef,
	})
	if err != nil {
		return err
	}

	log.Info().Str("cluster", id).Msg("Creating node group")
	ng, err := d.provider.CreateNodeGroup(ctx, id, cdef)
	if err != nil {
		return err
	}

	content, err := json.MarshalIndent(&ng, "", "    ")
	if err != nil {
		return err
	}

	fmt.Printf("NodeGroup:\n%s\n", string(content))

	log.Info().Str("cluster", id).Msg("Updating metadata with new nodes")
	var mns []metadata.Node
	cluster.Status = metadata.ClusterConnecting
	err = d.db.Update(ctx, func(tx *bolt.Tx) error {
		var err error
		tctx := metadata.WithTransactionContext(ctx, tx)
		cluster, err = d.db.UpdateCluster(tctx, cluster)
		if err != nil {
			return err
		}

		mns, err = d.db.CreateNodes(tctx, cluster.ID, ng.Nodes)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	nset := nodes.NewSet()
	for _, n := range mns {
		nset.Add(newNode(nil, n))
	}

	log.Info().Str("cluster", id).Msg("Connecting cluster")
	err = nodes.Connect(ctx, nset)
	if err != nil {
		return err
	}

	log.Info().Str("cluster", id).Msg("Updating cluster metadata")
	cluster.Status = metadata.ClusterCreated
	cluster, err = d.db.UpdateCluster(ctx, cluster)
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
	ctx := r.Context()
	id := vars["cluster"]

	cluster, err := d.db.GetCluster(ctx, id)
	if err != nil {
		return errors.Wrapf(err, "failed to get cluster %q", id)
	}

	if cluster.Status != metadata.ClusterDestroying {
		cluster.Status = metadata.ClusterDestroying
		cluster, err = d.db.UpdateCluster(ctx, cluster)
		if err != nil {
			return errors.Wrap(err, "failed to update cluster status to destroying")
		}
	}

	ns, err := d.db.ListNodes(ctx, cluster.ID)
	if err != nil {
		return errors.Wrap(err, "failed to list nodes")
	}

	ng := &p2plab.NodeGroup{
		ID:    cluster.ID,
		Nodes: ns,
	}

	err = d.provider.DestroyNodeGroup(ctx, ng)
	if err != nil {
		return errors.Wrap(err, "failed to destroy node group")
	}

	err = d.db.DeleteCluster(ctx, cluster.ID)
	if err != nil {
		return errors.Wrap(err, "failed to delete cluster metadata")
	}

	log.Info().Msg("Destroyed cluster")
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

	noReset := false
	if r.FormValue("no-reset") != "" {
		var err error
		noReset, err = strconv.ParseBool(r.FormValue("no-reset"))
		if err != nil {
			return err
		}
	}

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

	mns, err := d.db.ListNodes(ctx, cid)
	if err != nil {
		return err
	}

	nset := nodes.NewSet()
	for _, n := range mns {
		nset.Add(newNode(nil, n))
	}

	if !noReset {
		log.Info().Str("cluster", cid).Msg("Updating cluster")
		err = nodes.Update(ctx, nset, "")
		if err != nil {
			return err
		}

		log.Info().Str("cluster", cid).Msg("Connecting cluster")
		err = nodes.Connect(ctx, nset)
		if err != nil {
			return err
		}
	}

	log.Info().Str("cluster", cid).Str("scenario", sid).Msg("Creating scenario plan")
	plan, err := scenarios.Plan(ctx, filepath.Join(d.root, "transformers"), d.seeder, nset, scenario.Definition)
	if err != nil {
		return err
	}

	uuid := time.Now().Format(time.RFC3339Nano)
	benchmark := metadata.Benchmark{
		ID:       uuid,
		Status:   metadata.BenchmarkRunning,
		Cluster:  cluster,
		Scenario: scenario,
		Plan:     plan,
	}

	log.Info().Str("benchmark", benchmark.ID).Msg("Creating benchmark")
	benchmark, err = d.db.CreateBenchmark(ctx, benchmark)
	if err != nil {
		return err
	}

	seederAddr := fmt.Sprintf("%s/p2p/%s", d.seeder.Host().Addrs()[1], d.seeder.Host().ID())
	log.Info().Str("cluster", cid).Str("scenario", sid).Str("benchmark", benchmark.ID).Msg("Running scenario plan")
	err = scenarios.Run(ctx, nset, plan, seederAddr)
	if err != nil {
		log.Warn().Str("benchmark", benchmark.ID).Msgf("failed to run scenario plan: %s", err)
		return errors.Wrap(err, "failed to run scenario plan")
	}

	log.Info().Str("cluster", cid).Str("scenario", sid).Str("benchmark", benchmark.ID).Msg("Benchmark completed")
	benchmark.Status = metadata.BenchmarkDone
	benchmark, err = d.db.UpdateBenchmark(ctx, benchmark)
	if err != nil {
		log.Warn().Str("benchmark", benchmark.ID).Msgf("failed to update benchmark: %s", err)
		return errors.Wrap(err, "failed to update benchmark")
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

func (d *Labd) experimentsHandler(w http.ResponseWriter, r *http.Request) error {
	var err error
	switch r.Method {
	case "GET":
		err = d.listExperimentHandler(w, r)
	case "POST":
		err = d.createExperimentHandler(w, r)
	}
	return err
}

func (d *Labd) listExperimentHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("experiment/list")

	experiments, err := d.db.ListExperiments(r.Context())
	if err != nil {
		return err
	}

	return httputil.WriteJSON(w, &experiments)
}

func (d *Labd) createExperimentHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("experiment/create")
	return nil
}

func (d *Labd) getExperimentHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("experiment/get")

	vars := mux.Vars(r)
	experiment, err := d.db.GetExperiment(r.Context(), vars["experiment"])
	if err != nil {
		return err
	}

	return httputil.WriteJSON(w, &experiment)
}

func (d *Labd) cancelExperimentHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("experiment/cancel")

	vars := mux.Vars(r)
	_, err := d.db.GetExperiment(r.Context(), vars["experiment"])
	if err != nil {
		return err
	}

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
