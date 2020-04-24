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

package metadata

import (
	"context"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
)

const (
	schemaVersion = "v1"
)

type transactionKey struct{}

func WithTransactionContext(ctx context.Context, tx *bolt.Tx) context.Context {
	return context.WithValue(ctx, transactionKey{}, tx)
}

type DB interface {
	ClusterStore
	NodeStore
	ScenarioStore
	BuildStore
	ReportStore
	BenchmarkStore
	ExperimentStore

	View(ctx context.Context, fn func(*bolt.Tx) error) error

	Update(ctx context.Context, fn func(*bolt.Tx) error) error

	Close() error
}

type ClusterStore interface {
	GetCluster(ctx context.Context, id string) (Cluster, error)

	ListClusters(ctx context.Context) ([]Cluster, error)

	CreateCluster(ctx context.Context, cluster Cluster) (Cluster, error)

	UpdateCluster(ctx context.Context, cluster Cluster) (Cluster, error)

	LabelClusters(ctx context.Context, ids, adds, removes []string) ([]Cluster, error)

	DeleteCluster(ctx context.Context, id string) error
}

type NodeStore interface {
	GetNode(ctx context.Context, cluster, id string) (Node, error)

	ListNodes(ctx context.Context, cluster string) ([]Node, error)

	CreateNode(ctx context.Context, cluster string, node Node) (Node, error)

	CreateNodes(ctx context.Context, cluster string, node []Node) ([]Node, error)

	UpdateNode(ctx context.Context, cluster string, node Node) (Node, error)

	LabelNodes(ctx context.Context, cluster string, ids, adds, removes []string) ([]Node, error)
}

type ScenarioStore interface {
	GetScenario(ctx context.Context, id string) (Scenario, error)

	ListScenarios(ctx context.Context) ([]Scenario, error)

	CreateScenario(ctx context.Context, scenario Scenario) (Scenario, error)

	UpdateScenario(ctx context.Context, scenario Scenario) (Scenario, error)

	LabelScenarios(ctx context.Context, ids, adds, removes []string) ([]Scenario, error)

	DeleteScenarios(ctx context.Context, ids ...string) error
}

type BuildStore interface {
	GetBuild(ctx context.Context, id string) (Build, error)

	ListBuilds(ctx context.Context) ([]Build, error)

	CreateBuild(ctx context.Context, build Build) (Build, error)

	DeleteBuild(ctx context.Context, id string) error
}

type ReportStore interface {
	GetReport(ctx context.Context, id string) (Report, error)

	CreateReport(ctx context.Context, id string, report Report) error
}

type BenchmarkStore interface {
	GetBenchmark(ctx context.Context, id string) (Benchmark, error)

	ListBenchmarks(ctx context.Context) ([]Benchmark, error)

	CreateBenchmark(ctx context.Context, benchmark Benchmark) (Benchmark, error)

	UpdateBenchmark(ctx context.Context, benchmark Benchmark) (Benchmark, error)

	LabelBenchmarks(ctx context.Context, ids, adds, removes []string) ([]Benchmark, error)

	DeleteBenchmarks(ctx context.Context, ids ...string) error
}

type ExperimentStore interface {
	GetExperiment(ctx context.Context, id string) (Experiment, error)

	ListExperiments(ctx context.Context) ([]Experiment, error)

	CreateExperiment(ctx context.Context, experiment Experiment) (Experiment, error)

	UpdateExperiment(ctx context.Context, experiment Experiment) (Experiment, error)

	LabelExperiments(ctx context.Context, ids, adds, removes []string) ([]Experiment, error)

	DeleteExperiment(ctx context.Context, id string) error
}

type db struct {
	boltdb *bolt.DB
}

func NewDB(root string) (DB, error) {
	if _, err := os.Stat(root); os.IsNotExist(err) {
		if err := os.Mkdir(root, os.FileMode(0740)); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	path := filepath.Join(root, "meta.db")
	boltdb, err := bolt.Open(path, 0644, nil)
	if err != nil {
		return nil, err
	}

	return &db{boltdb}, nil
}

func (m *db) Close() error {
	return m.boltdb.Close()
}

func (m *db) View(ctx context.Context, fn func(*bolt.Tx) error) error {
	tx, ok := ctx.Value(transactionKey{}).(*bolt.Tx)
	if !ok {
		return m.boltdb.View(fn)
	}
	return fn(tx)
}

func (m *db) Update(ctx context.Context, fn func(*bolt.Tx) error) error {
	tx, ok := ctx.Value(transactionKey{}).(*bolt.Tx)
	if !ok {
		return m.boltdb.Update(fn)
	} else if !tx.Writable() {
		return errors.Wrap(bolt.ErrTxNotWritable, "unable to use transaction from context")
	}
	return fn(tx)
}

type field struct {
	key   []byte
	value []byte
}
