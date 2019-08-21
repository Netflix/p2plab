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
	"time"

	"github.com/Netflix/p2plab/errdefs"
	cid "github.com/ipfs/go-cid"
	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
)

type Benchmark struct {
	ID string

	Status BenchmarkStatus

	Cluster  Cluster
	Scenario Scenario
	Plan     ScenarioPlan

	Labels []string

	CreatedAt, UpdatedAt time.Time
}

// BenchmarkStatus is the current status of a benchmark.
type BenchmarkStatus string

var (
	BenchmarkPlanning BenchmarkStatus = "planning"

	BenchmarkRunning BenchmarkStatus = "running"

	BenchmarkDone BenchmarkStatus = "done"

	BenchmarkError BenchmarkStatus = "error"
)

type ScenarioPlan struct {
	Objects map[string]cid.Cid

	Seed ScenarioStage

	Benchmark ScenarioStage
}

type ScenarioStage map[string]Task

type Task struct {
	Type TaskType

	Subject string
}

type TaskType string

var (
	TaskUpdate     TaskType = "update"
	TaskGet        TaskType = "get"
	TaskConnect    TaskType = "connect"
	TaskDisconnect TaskType = "disconnect"
)

func (m *db) GetBenchmark(ctx context.Context, id string) (Benchmark, error) {
	var benchmark Benchmark

	err := m.View(ctx, func(tx *bolt.Tx) error {
		bkt := getBenchmarksBucket(tx)
		if bkt == nil {
			return errors.Wrapf(errdefs.ErrNotFound, "benchmark %q", id)
		}

		cbkt := bkt.Bucket([]byte(id))
		if cbkt == nil {
			return errors.Wrapf(errdefs.ErrNotFound, "benchmark %q", id)
		}

		benchmark.ID = id
		err := readBenchmark(cbkt, &benchmark)
		if err != nil {
			return errors.Wrapf(err, "benchmark %q", id)
		}

		return nil
	})
	if err != nil {
		return Benchmark{}, err
	}

	return benchmark, nil
}

func (m *db) ListBenchmarks(ctx context.Context) ([]Benchmark, error) {
	var benchmarks []Benchmark
	err := m.View(ctx, func(tx *bolt.Tx) error {
		bkt := getBenchmarksBucket(tx)
		if bkt == nil {
			return nil
		}

		return bkt.ForEach(func(k, v []byte) error {
			var (
				benchmark = Benchmark{
					ID: string(k),
				}
				cbkt = bkt.Bucket(k)
			)

			err := readBenchmark(cbkt, &benchmark)
			if err != nil {
				return err
			}

			benchmarks = append(benchmarks, benchmark)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	return benchmarks, nil
}

func (m *db) CreateBenchmark(ctx context.Context, benchmark Benchmark) (Benchmark, error) {
	err := m.Update(ctx, func(tx *bolt.Tx) error {
		bkt, err := createBenchmarksBucket(tx)
		if err != nil {
			return err
		}

		cbkt, err := bkt.CreateBucket([]byte(benchmark.ID))
		if err != nil {
			if err != bolt.ErrBucketExists {
				return err
			}

			return errors.Wrapf(errdefs.ErrAlreadyExists, "benchmark %q", benchmark.ID)
		}

		benchmark.CreatedAt = time.Now().UTC()
		benchmark.UpdatedAt = benchmark.CreatedAt
		return writeBenchmark(cbkt, &benchmark)
	})
	if err != nil {
		return Benchmark{}, err
	}
	return benchmark, err
}

func (m *db) UpdateBenchmark(ctx context.Context, benchmark Benchmark) (Benchmark, error) {
	if benchmark.ID == "" {
		return Benchmark{}, errors.Wrapf(errdefs.ErrInvalidArgument, "benchmark id required for update")
	}

	err := m.Update(ctx, func(tx *bolt.Tx) error {
		bkt, err := createBenchmarksBucket(tx)
		if err != nil {
			return err
		}

		cbkt := bkt.Bucket([]byte(benchmark.ID))
		if cbkt == nil {
			return errors.Wrapf(errdefs.ErrNotFound, "benchmark %q", benchmark.ID)
		}

		benchmark.UpdatedAt = time.Now().UTC()
		return writeBenchmark(cbkt, &benchmark)
	})
	if err != nil {
		return Benchmark{}, err
	}

	return benchmark, nil
}

func (m *db) LabelBenchmarks(ctx context.Context, ids, adds, removes []string) ([]Benchmark, error) {
	var benchmarks []Benchmark
	err := m.Update(ctx, func(tx *bolt.Tx) error {
		bkt, err := createBenchmarksBucket(tx)
		if err != nil {
			return err
		}

		err = batchUpdateLabels(bkt, ids, adds, removes, func(ibkt *bolt.Bucket, id string, labels []string) error {
			var benchmark Benchmark
			benchmark.ID = id
			err = readBenchmark(ibkt, &benchmark)
			if err != nil {
				return err
			}

			benchmark.Labels = labels
			benchmark.UpdatedAt = time.Now().UTC()

			err = writeBenchmark(ibkt, &benchmark)
			if err != nil {
				return err
			}
			benchmarks = append(benchmarks, benchmark)
			return nil
		})
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return benchmarks, nil
}

func (m *db) DeleteBenchmarks(ctx context.Context, ids ...string) error {
	return m.Update(ctx, func(tx *bolt.Tx) error {
		bkt := getBenchmarksBucket(tx)
		if bkt == nil {
			return nil
		}

		for _, id := range ids {
			err := bkt.DeleteBucket([]byte(id))
			if err != nil {
				if err == bolt.ErrBucketNotFound {
					return errors.Wrapf(errdefs.ErrNotFound, "benchmark %q", id)
				}
				return err
			}
		}

		return nil
	})
}

func readBenchmark(bkt *bolt.Bucket, benchmark *Benchmark) error {
	err := ReadTimestamps(bkt, &benchmark.CreatedAt, &benchmark.UpdatedAt)
	if err != nil {
		return err
	}

	cbkt := bkt.Bucket(bucketKeyCluster)
	if cbkt != nil {
		err = readCluster(cbkt, &benchmark.Cluster)
		if err != nil {
			return err
		}
	}

	sbkt := bkt.Bucket(bucketKeyScenario)
	if sbkt != nil {
		err = readScenario(sbkt, &benchmark.Scenario)
		if err != nil {
			return err
		}
	}

	pbkt := bkt.Bucket(bucketKeyPlan)
	if pbkt != nil {
		err = readPlan(pbkt, &benchmark.Plan)
		if err != nil {
			return err
		}
	}

	benchmark.Labels, err = readLabels(bkt)
	if err != nil {
		return err
	}

	return bkt.ForEach(func(k, v []byte) error {
		if v == nil {
			return nil
		}

		switch string(k) {
		case string(bucketKeyID):
			benchmark.ID = string(v)
		case string(bucketKeyStatus):
			benchmark.Status = BenchmarkStatus(v)
		}

		return nil
	})
}

func readPlan(bkt *bolt.Bucket, plan *ScenarioPlan) error {
	m, err := readMap(bkt, bucketKeyObjects)
	if err != nil {
		return nil
	}

	if m != nil {
		objects := make(map[string]cid.Cid)
		for k, v := range m {
			objects[k], err = cid.Parse(v)
			if err != nil {
				return err
			}
		}
		plan.Objects = objects
	}

	plan.Seed, err = readTaskMap(bkt, bucketKeySeed)
	if err != nil {
		return nil
	}

	plan.Benchmark, err = readTaskMap(bkt, bucketKeyBenchmark)
	if err != nil {
		return nil
	}

	return nil
}

func readTaskMap(bkt *bolt.Bucket, name []byte) (map[string]Task, error) {
	tbkt := bkt.Bucket(name)
	if tbkt == nil {
		return nil, nil
	}

	tasks := make(map[string]Task)
	err := tbkt.ForEach(func(id, v []byte) error {
		ibkt := tbkt.Bucket(id)
		if ibkt == nil {
			return nil
		}

		var task Task
		err := ibkt.ForEach(func(k, v []byte) error {
			switch string(k) {
			case string(bucketKeyType):
				task.Type = TaskType(v)
			case string(bucketKeySubject):
				task.Subject = string(v)
			}
			return nil
		})
		if err != nil {
			return err
		}

		tasks[string(id)] = task
		return nil
	})
	if err != nil {
		return nil, err
	}

	return tasks, nil
}

func writeBenchmark(bkt *bolt.Bucket, benchmark *Benchmark) error {
	err := WriteTimestamps(bkt, benchmark.CreatedAt, benchmark.UpdatedAt)
	if err != nil {
		return err
	}

	cbkt := bkt.Bucket(bucketKeyCluster)
	if cbkt != nil {
		err = bkt.DeleteBucket(bucketKeyCluster)
		if err != nil {
			return err
		}
	}

	cbkt, err = bkt.CreateBucket(bucketKeyCluster)
	if err != nil {
		return err
	}

	err = writeCluster(cbkt, &benchmark.Cluster)
	if err != nil {
		return err
	}

	sbkt := bkt.Bucket(bucketKeyScenario)
	if sbkt != nil {
		err = bkt.DeleteBucket(bucketKeyScenario)
		if err != nil {
			return err
		}
	}

	sbkt, err = bkt.CreateBucket(bucketKeyScenario)
	if err != nil {
		return err
	}

	err = writeScenario(sbkt, &benchmark.Scenario)
	if err != nil {
		return err
	}

	pbkt := bkt.Bucket(bucketKeyPlan)
	if pbkt != nil {
		err = bkt.DeleteBucket(bucketKeyPlan)
		if err != nil {
			return err
		}
	}

	pbkt, err = bkt.CreateBucket(bucketKeyPlan)
	if err != nil {
		return err
	}

	err = writePlan(pbkt, &benchmark.Plan)
	if err != nil {
		return err
	}

	err = writeLabels(bkt, benchmark.Labels)
	if err != nil {
		return err
	}

	for _, f := range []field{
		{bucketKeyID, []byte(benchmark.ID)},
		{bucketKeyStatus, []byte(benchmark.Status)},
	} {
		err = bkt.Put(f.key, f.value)
		if err != nil {
			return err
		}
	}

	return nil
}

func writePlan(bkt *bolt.Bucket, plan *ScenarioPlan) error {
	obkt := bkt.Bucket(bucketKeyObjects)
	if obkt != nil {
		err := bkt.DeleteBucket(bucketKeyObjects)
		if err != nil {
			return err
		}
	}

	var err error
	obkt, err = bkt.CreateBucket(bucketKeyObjects)
	if err != nil {
		return err
	}

	m := make(map[string]string)
	for k, v := range plan.Objects {
		m[k] = v.String()
	}

	err = writeMap(bkt, bucketKeyObjects, m)
	if err != nil {
		return err
	}

	err = writeTaskMap(bkt, bucketKeySeed, plan.Seed)
	if err != nil {
		return err
	}

	err = writeTaskMap(bkt, bucketKeyBenchmark, plan.Benchmark)
	if err != nil {
		return err
	}

	return nil
}

func writeTaskMap(bkt *bolt.Bucket, name []byte, stage map[string]Task) error {
	mbkt := bkt.Bucket(name)
	if mbkt != nil {
		err := bkt.DeleteBucket(name)
		if err != nil {
			return err
		}
	}

	if len(stage) == 0 {
		return nil
	}

	var err error
	mbkt, err = bkt.CreateBucket(name)
	if err != nil {
		return err
	}

	for id, task := range stage {
		tbkt, err := mbkt.CreateBucket([]byte(id))
		if err != nil {
			return err
		}

		for _, f := range []field{
			{bucketKeyType, []byte(task.Type)},
			{bucketKeySubject, []byte(task.Subject)},
		} {
			err = tbkt.Put(f.key, f.value)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
