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

	Cluster  Cluster
	Scenario Scenario
	Plan     ScenarioPlan

	CreatedAt, UpdatedAt time.Time
}

type ScenarioPlan struct {
	Objects map[string]cid.Cid

	Seed map[string]Task

	Benchmark map[string]Task
}

type Task struct {
	Type   TaskType
	Target string
}

type TaskType string

var (
	TaskUpdate TaskType = "update"
	TaskGet    TaskType = "get"
)

func (m *DB) GetBenchmark(ctx context.Context, id string) (Benchmark, error) {
	var benchmark Benchmark

	err := m.View(func(tx *bolt.Tx) error {
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

func (m *DB) ListBenchmarks(ctx context.Context) ([]Benchmark, error) {
	var benchmarks []Benchmark
	err := m.View(func(tx *bolt.Tx) error {
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

func (m *DB) CreateBenchmark(ctx context.Context, benchmark Benchmark) (Benchmark, error) {
	err := m.Update(func(tx *bolt.Tx) error {
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

func (m *DB) UpdateBenchmark(ctx context.Context, benchmark Benchmark) (Benchmark, error) {
	if benchmark.ID == "" {
		return Benchmark{}, errors.Wrapf(errdefs.ErrInvalidArgument, "benchmark id required for update")
	}

	err := m.Update(func(tx *bolt.Tx) error {
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

func (m *DB) DeleteBenchmark(ctx context.Context, id string) error {
	return m.Update(func(tx *bolt.Tx) error {
		bkt := getBenchmarksBucket(tx)
		if bkt == nil {
			return errors.Wrapf(errdefs.ErrNotFound, "benchmark %q", id)
		}

		err := bkt.DeleteBucket([]byte(id))
		if err == bolt.ErrBucketNotFound {
			return errors.Wrapf(errdefs.ErrNotFound, "benchmark %q", id)
		}
		return err
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

	return bkt.ForEach(func(k, v []byte) error {
		if v == nil {
			return nil
		}

		switch string(k) {
		case string(bucketKeyID):
			benchmark.ID = string(v)
		}

		return nil
	})
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

	for _, f := range []field{
		{bucketKeyID, []byte(benchmark.ID)},
	} {
		err = bkt.Put(f.key, f.value)
		if err != nil {
			return err
		}
	}

	return nil
}
