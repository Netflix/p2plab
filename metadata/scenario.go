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
	"strconv"
	"time"

	"github.com/Netflix/p2plab/errdefs"
	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
)

type Scenario struct {
	ID string

	Definition ScenarioDefinition

	Labels []string

	CreatedAt, UpdatedAt time.Time
}

// ScenarioDefinition defines a scenario.
type ScenarioDefinition struct {
	Objects map[string]ObjectDefinition `json:"objects,omitempty"`

	// Seed map a query to an action. Queries are executed in parallel to seed
	// a cluster with initial data before running the benchmark.
	Seed map[string]string `json:"seed,omitempty"`

	// Benchmark maps a query to an action. Queries are executed in parallel
	// during the benchmark and metrics are collected during this stage.
	Benchmark map[string]string `json:"benchmark,omitempty"`
}

// ObjectDefinition define a type of data that will be distributed during the
// benchmark. The definition also specify options on how the data is converted
// into IPFS datastructures.
type ObjectDefinition struct {
	// Type specifies what type is the source of the data and how the data is
	// retrieved. Types must be one of the following: ["oci-image"].
	Type string `json:"type"`

	Source string `json:"source"`

	// Layout specify how the DAG is shaped and constructed over the IPLD blocks.
	Layout string `json:"layout"`

	// Chunker specify which chunking algorithm to use to chunk the data into IPLD
	// blocks.
	Chunker string `json:"chunker"`

	RawLeaves bool `json:"rawLeaves"`

	HashFunc string `json:"hashFunc"`

	MaxLinks int `json:"maxLinks"`
}

// ObjectType is the type of data retrieved.
type ObjectType string

var (
	// ObjectContainerImage indicates that the object is an OCI image.
	ObjectContainerImage ObjectType = "oci-image"
)

func (m *db) GetScenario(ctx context.Context, id string) (Scenario, error) {
	var scenario Scenario

	err := m.View(ctx, func(tx *bolt.Tx) error {
		bkt := getScenariosBucket(tx)
		if bkt == nil {
			return errors.Wrapf(errdefs.ErrNotFound, "scenario %q", id)
		}

		sbkt := bkt.Bucket([]byte(id))
		if sbkt == nil {
			return errors.Wrapf(errdefs.ErrNotFound, "scenario %q", id)
		}

		scenario.ID = id
		err := readScenario(sbkt, &scenario)
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

func (m *db) ListScenarios(ctx context.Context) ([]Scenario, error) {
	var scenarios []Scenario
	err := m.View(ctx, func(tx *bolt.Tx) error {
		bkt := getScenariosBucket(tx)
		if bkt == nil {
			return nil
		}

		return bkt.ForEach(func(k, v []byte) error {
			var (
				scenario = Scenario{
					ID: string(k),
				}
				sbkt = bkt.Bucket(k)
			)

			err := readScenario(sbkt, &scenario)
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

func (m *db) CreateScenario(ctx context.Context, scenario Scenario) (Scenario, error) {
	err := m.Update(ctx, func(tx *bolt.Tx) error {
		bkt, err := createScenariosBucket(tx)
		if err != nil {
			return err
		}

		sbkt, err := bkt.CreateBucket([]byte(scenario.ID))
		if err != nil {
			if err != bolt.ErrBucketExists {
				return err
			}

			return errors.Wrapf(errdefs.ErrAlreadyExists, "scenario %q", scenario.ID)
		}

		scenario.CreatedAt = time.Now().UTC()
		scenario.UpdatedAt = scenario.CreatedAt
		return writeScenario(sbkt, &scenario)
	})
	if err != nil {
		return Scenario{}, err
	}
	return scenario, err
}

func (m *db) UpdateScenario(ctx context.Context, scenario Scenario) (Scenario, error) {
	if scenario.ID == "" {
		return Scenario{}, errors.Wrapf(errdefs.ErrInvalidArgument, "scenario id required for update")
	}

	err := m.Update(ctx, func(tx *bolt.Tx) error {
		bkt, err := createScenariosBucket(tx)
		if err != nil {
			return err
		}

		sbkt := bkt.Bucket([]byte(scenario.ID))
		if sbkt == nil {
			return errors.Wrapf(errdefs.ErrNotFound, "scenario %q", scenario.ID)
		}

		scenario.UpdatedAt = time.Now().UTC()
		return writeScenario(sbkt, &scenario)
	})
	if err != nil {
		return Scenario{}, err
	}

	return scenario, nil
}

func (m *db) LabelScenarios(ctx context.Context, ids, adds, removes []string) ([]Scenario, error) {
	var scenarios []Scenario
	err := m.Update(ctx, func(tx *bolt.Tx) error {
		bkt, err := createScenariosBucket(tx)
		if err != nil {
			return err
		}

		err = batchUpdateLabels(bkt, ids, adds, removes, func(ibkt *bolt.Bucket, id string, labels []string) error {
			var scenario Scenario
			scenario.ID = id
			err = readScenario(ibkt, &scenario)
			if err != nil {
				return err
			}

			scenario.Labels = labels
			scenario.UpdatedAt = time.Now().UTC()

			err = writeScenario(ibkt, &scenario)
			if err != nil {
				return err
			}
			scenarios = append(scenarios, scenario)
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

	return scenarios, nil
}

func (m *db) DeleteScenarios(ctx context.Context, ids ...string) error {
	return m.Update(ctx, func(tx *bolt.Tx) error {
		bkt := getScenariosBucket(tx)
		if bkt == nil {
			return nil
		}

		for _, id := range ids {
			err := bkt.DeleteBucket([]byte(id))
			if err != nil {
				if err == bolt.ErrBucketNotFound {
					return errors.Wrapf(errdefs.ErrNotFound, "scenario %q", id)
				}
				return err
			}
		}

		return nil
	})
}

func readScenario(bkt *bolt.Bucket, scenario *Scenario) error {
	err := ReadTimestamps(bkt, &scenario.CreatedAt, &scenario.UpdatedAt)
	if err != nil {
		return err
	}

	scenario.Definition, err = readScenarioDefinition(bkt)
	if err != nil {
		return err
	}

	scenario.Labels, err = readLabels(bkt)
	if err != nil {
		return err
	}

	return bkt.ForEach(func(k, v []byte) error {
		if v == nil {
			return nil
		}

		switch string(k) {
		case string(bucketKeyID):
			scenario.ID = string(v)
		}

		return nil
	})
}

func readScenarioDefinition(bkt *bolt.Bucket) (ScenarioDefinition, error) {
	var sdef ScenarioDefinition

	dbkt := bkt.Bucket(bucketKeyDefinition)
	if dbkt == nil {
		return sdef, nil
	}

	var err error
	sdef.Objects, err = readObjects(dbkt)
	if err != nil {
		return sdef, err
	}

	sdef.Seed, err = readMap(dbkt, bucketKeySeed)
	if err != nil {
		return sdef, err
	}

	sdef.Benchmark, err = readMap(dbkt, bucketKeyBenchmark)
	if err != nil {
		return sdef, err
	}

	return sdef, nil
}

func writeScenario(bkt *bolt.Bucket, scenario *Scenario) error {
	err := WriteTimestamps(bkt, scenario.CreatedAt, scenario.UpdatedAt)
	if err != nil {
		return err
	}

	err = writeScenarioDefinition(bkt, scenario.Definition)
	if err != nil {
		return err
	}

	err = writeLabels(bkt, scenario.Labels)
	if err != nil {
		return err
	}

	for _, f := range []field{
		{bucketKeyID, []byte(scenario.ID)},
	} {
		err = bkt.Put(f.key, f.value)
		if err != nil {
			return err
		}
	}

	return nil
}

func writeScenarioDefinition(bkt *bolt.Bucket, sdef ScenarioDefinition) error {
	dbkt := bkt.Bucket(bucketKeyDefinition)
	if dbkt != nil {
		err := bkt.DeleteBucket(bucketKeyDefinition)
		if err != nil {
			return err
		}
	}

	dbkt, err := bkt.CreateBucket(bucketKeyDefinition)
	if err != nil {
		return err
	}

	err = writeObjects(dbkt, sdef.Objects)
	if err != nil {
		return err
	}

	err = writeMap(dbkt, bucketKeySeed, sdef.Seed)
	if err != nil {
		return err
	}

	err = writeMap(dbkt, bucketKeyBenchmark, sdef.Benchmark)
	if err != nil {
		return err
	}

	return nil
}

func readObjects(bkt *bolt.Bucket) (map[string]ObjectDefinition, error) {
	obkt := bkt.Bucket(bucketKeyObjects)
	if obkt == nil {
		return nil, nil
	}

	objects := map[string]ObjectDefinition{}
	err := obkt.ForEach(func(name, v []byte) error {
		nbkt := obkt.Bucket(name)
		if nbkt == nil {
			return nil
		}

		var object ObjectDefinition
		err := nbkt.ForEach(func(k, v []byte) error {
			switch string(k) {
			case string(bucketKeyType):
				object.Type = string(v)
			case string(bucketKeySource):
				object.Source = string(v)
			case string(bucketKeyLayout):
				object.Layout = string(v)
			case string(bucketKeyChunker):
				object.Chunker = string(v)
			case string(bucketKeyRawLeaves):
				object.RawLeaves, _ = strconv.ParseBool(string(v))
			case string(bucketKeyHashFunc):
				object.HashFunc = string(v)
			case string(bucketKeyMaxLinks):
				object.MaxLinks, _ = strconv.Atoi(string(v))
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

func writeObjects(bkt *bolt.Bucket, objects map[string]ObjectDefinition) error {
	obkt, err := RecreateBucket(bkt, bucketKeyObjects)
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
			{bucketKeySource, []byte(object.Source)},
			{bucketKeyLayout, []byte(object.Layout)},
			{bucketKeyChunker, []byte(object.Chunker)},
			{bucketKeyRawLeaves, []byte(strconv.FormatBool(object.RawLeaves))},
			{bucketKeyHashFunc, []byte(object.HashFunc)},
			{bucketKeyMaxLinks, []byte(strconv.Itoa(object.MaxLinks))},
		} {
			err = nbkt.Put(f.key, f.value)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
