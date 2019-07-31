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
	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
)

type Scenario struct {
	ID string

	Objects map[string]Object

	Seed map[string]string

	Benchmark map[string]string

	CreatedAt, UpdatedAt time.Time
}

type Object struct {
	Type      string
	Reference string
}

func (m *DB) GetScenario(ctx context.Context, id string) (Scenario, error) {
	var scenario Scenario

	err := m.View(func(tx *bolt.Tx) error {
		bkt := getScenariosBucket(tx)
		if bkt == nil {
			return errors.Wrapf(errdefs.ErrNotFound, "scenario %q", id)
		}

		cbkt := bkt.Bucket([]byte(id))
		if cbkt == nil {
			return errors.Wrapf(errdefs.ErrNotFound, "scenario %q", id)
		}

		scenario.ID = id
		err := readScenario(cbkt, &scenario)
		if err != nil {
			return errors.Wrapf(err, "scenario %q", id)
		}

		return nil
	})
	if err != nil {
		return Scenario{}, err
	}

	return scenario, nil
}

func (m *DB) ListScenarios(ctx context.Context) ([]Scenario, error) {
	var scenarios []Scenario
	err := m.View(func(tx *bolt.Tx) error {
		bkt := getScenariosBucket(tx)
		if bkt == nil {
			return nil
		}

		return bkt.ForEach(func(k, v []byte) error {
			var (
				scenario = Scenario{
					ID: string(k),
				}
				cbkt = bkt.Bucket(k)
			)

			err := readScenario(cbkt, &scenario)
			if err != nil {
				return err
			}

			scenarios = append(scenarios, scenario)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	return scenarios, nil
}

func (m *DB) CreateScenario(ctx context.Context, scenario Scenario) (Scenario, error) {
	err := m.Update(func(tx *bolt.Tx) error {
		bkt, err := createScenariosBucket(tx)
		if err != nil {
			return err
		}

		cbkt, err := bkt.CreateBucket([]byte(scenario.ID))
		if err != nil {
			if err != bolt.ErrBucketExists {
				return err
			}

			return errors.Wrapf(errdefs.ErrAlreadyExists, "scenario %q", scenario.ID)
		}

		scenario.CreatedAt = time.Now().UTC()
		scenario.UpdatedAt = scenario.CreatedAt
		return writeScenario(cbkt, &scenario)
	})
	if err != nil {
		return Scenario{}, err
	}
	return scenario, err
}

func (m *DB) UpdateScenario(ctx context.Context, scenario Scenario) (Scenario, error) {
	if scenario.ID == "" {
		return Scenario{}, errors.Wrapf(errdefs.ErrInvalidArgument, "scenario id required for update")
	}

	err := m.Update(func(tx *bolt.Tx) error {
		bkt, err := createScenariosBucket(tx)
		if err != nil {
			return err
		}

		cbkt := bkt.Bucket([]byte(scenario.ID))
		if cbkt == nil {
			return errors.Wrapf(errdefs.ErrNotFound, "scenario %q", scenario.ID)
		}

		scenario.UpdatedAt = time.Now().UTC()
		return writeScenario(cbkt, &scenario)
	})
	if err != nil {
		return Scenario{}, err
	}

	return scenario, nil
}

func (m *DB) DeleteScenario(ctx context.Context, id string) error {
	return m.Update(func(tx *bolt.Tx) error {
		bkt := getScenariosBucket(tx)
		if bkt == nil {
			return errors.Wrapf(errdefs.ErrNotFound, "scenario %q", id)
		}

		err := bkt.DeleteBucket([]byte(id))
		if err == bolt.ErrBucketNotFound {
			return errors.Wrapf(errdefs.ErrNotFound, "scenario %q", id)
		}
		return err
	})
}

func readScenario(bkt *bolt.Bucket, scenario *Scenario) error {
	err := ReadTimestamps(bkt, &scenario.CreatedAt, &scenario.UpdatedAt)
	if err != nil {
		return err
	}

	objects, err := readObjects(bkt)
	if err != nil {
		return err
	}
	scenario.Objects = objects

	seed, err := readMap(bkt, bucketKeySeed)
	if err != nil {
		return err
	}
	scenario.Seed = seed

	benchmark, err := readMap(bkt, bucketKeyBenchmark)
	if err != nil {
		return err
	}
	scenario.Benchmark = benchmark

	return nil
}

func writeScenario(bkt *bolt.Bucket, scenario *Scenario) error {
	err := WriteTimestamps(bkt, scenario.CreatedAt, scenario.UpdatedAt)
	if err != nil {
		return err
	}

	err = writeObjects(bkt, scenario.Objects)
	if err != nil {
		return err
	}

	err = writeMap(bkt, bucketKeySeed, scenario.Seed)
	if err != nil {
		return err
	}

	err = writeMap(bkt, bucketKeyBenchmark, scenario.Benchmark)
	if err != nil {
		return err
	}

	return nil
}

func readObjects(bkt *bolt.Bucket) (map[string]Object, error) {
	obkt := bkt.Bucket(bucketKeyObjects)
	if obkt == nil {
		return nil, nil
	}

	objects := map[string]Object{}
	err := obkt.ForEach(func(name, v []byte) error {
		if v == nil {
			return nil
		}

		nbkt := obkt.Bucket(name)
		if nbkt == nil {
			return nil
		}

		var object Object
		err := nbkt.ForEach(func(k, v []byte) error {
			switch string(k) {
			case string(bucketKeyType):
				object.Type = string(v)
			case string(bucketKeyReference):
				object.Reference = string(v)
			}
			return nil
		})
		if err != nil {
			return err
		}

		objects[string(name)] = object
		return nil
	})
	if err != nil {
		return nil, err
	}

	return objects, nil
}

func writeObjects(bkt *bolt.Bucket, objects map[string]Object) error {
	obkt := bkt.Bucket(bucketKeyObjects)
	if obkt != nil {
		err := bkt.DeleteBucket(bucketKeyObjects)
		if err != nil {
			return err
		}
	}

	if len(objects) == 0 {
		return nil
	}

	var err error
	obkt, err = bkt.CreateBucket(bucketKeyObjects)
	if err != nil {
		return err
	}

	for name, object := range objects {
		nbkt, err := obkt.CreateBucket([]byte(name))
		if err != nil {
			return err
		}

		for _, f := range []field{
			{bucketKeyType, []byte(object.Type)},
			{bucketKeyReference, []byte(object.Reference)},
		} {
			err = nbkt.Put(f.key, f.value)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func readMap(bkt *bolt.Bucket, name []byte) (map[string]string, error) {
	mbkt := bkt.Bucket(name)
	if mbkt == nil {
		return nil, nil
	}

	m := map[string]string{}
	err := mbkt.ForEach(func(k, v []byte) error {
		m[string(k)] = string(v)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return m, nil
}

func writeMap(bkt *bolt.Bucket, name []byte, m map[string]string) error {
	// Remove existing map to prevent merging.
	mbkt := bkt.Bucket(name)
	if mbkt != nil {
		err := bkt.DeleteBucket(name)
		if err != nil {
			return err
		}
	}

	if len(m) == 0 {
		return nil
	}

	var err error
	mbkt, err = bkt.CreateBucket(name)
	if err != nil {
		return err
	}

	for k, v := range m {
		if v == "" {
			delete(m, k)
			continue
		}

		err := mbkt.Put([]byte(k), []byte(v))
		if err != nil {
			return errors.Wrapf(err, "failed to set key value %q=%q", k, v)
		}
	}

	return nil
}
