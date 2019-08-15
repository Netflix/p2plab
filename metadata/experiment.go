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

type Experiment struct {
	ID string

	Definition ExperimentDefinition

	Labels []string

	CreatedAt, UpdatedAt time.Time
}

// ExperimentDefinition defines an experiment.
type ExperimentDefinition struct {
}

func (m *DB) GetExperiment(ctx context.Context, id string) (Experiment, error) {
	var experiment Experiment

	err := m.View(ctx, func(tx *bolt.Tx) error {
		bkt := getExperimentsBucket(tx)
		if bkt == nil {
			return errors.Wrapf(errdefs.ErrNotFound, "experiment %q", id)
		}

		ebkt := bkt.Bucket([]byte(id))
		if ebkt == nil {
			return errors.Wrapf(errdefs.ErrNotFound, "experiment %q", id)
		}

		experiment.ID = id
		err := readExperiment(ebkt, &experiment)
		if err != nil {
			return errors.Wrapf(err, "experiment %q", id)
		}

		return nil
	})
	if err != nil {
		return Experiment{}, err
	}

	return experiment, nil
}

func (m *DB) ListExperiments(ctx context.Context) ([]Experiment, error) {
	var experiments []Experiment
	err := m.View(ctx, func(tx *bolt.Tx) error {
		bkt := getExperimentsBucket(tx)
		if bkt == nil {
			return nil
		}

		return bkt.ForEach(func(k, v []byte) error {
			var (
				experiment = Experiment{
					ID: string(k),
				}
				ebkt = bkt.Bucket(k)
			)

			err := readExperiment(ebkt, &experiment)
			if err != nil {
				return err
			}

			experiments = append(experiments, experiment)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	return experiments, nil
}

func (m *DB) CreateExperiment(ctx context.Context, experiment Experiment) (Experiment, error) {
	err := m.Update(ctx, func(tx *bolt.Tx) error {
		bkt, err := createExperimentsBucket(tx)
		if err != nil {
			return err
		}

		ebkt, err := bkt.CreateBucket([]byte(experiment.ID))
		if err != nil {
			if err != bolt.ErrBucketExists {
				return err
			}

			return errors.Wrapf(errdefs.ErrAlreadyExists, "experiment %q", experiment.ID)
		}

		experiment.CreatedAt = time.Now().UTC()
		experiment.UpdatedAt = experiment.CreatedAt
		return writeExperiment(ebkt, &experiment)
	})
	if err != nil {
		return Experiment{}, err
	}
	return experiment, err
}

func (m *DB) UpdateExperiment(ctx context.Context, experiment Experiment) (Experiment, error) {
	if experiment.ID == "" {
		return Experiment{}, errors.Wrapf(errdefs.ErrInvalidArgument, "experiment id required for update")
	}

	err := m.Update(ctx, func(tx *bolt.Tx) error {
		bkt, err := createExperimentsBucket(tx)
		if err != nil {
			return err
		}

		ebkt := bkt.Bucket([]byte(experiment.ID))
		if ebkt == nil {
			return errors.Wrapf(errdefs.ErrNotFound, "experiment %q", experiment.ID)
		}

		experiment.UpdatedAt = time.Now().UTC()
		return writeExperiment(ebkt, &experiment)
	})
	if err != nil {
		return Experiment{}, err
	}

	return experiment, nil
}

func (m *DB) DeleteExperiment(ctx context.Context, id string) error {
	return m.Update(ctx, func(tx *bolt.Tx) error {
		bkt := getExperimentsBucket(tx)
		if bkt == nil {
			return errors.Wrapf(errdefs.ErrNotFound, "experiment %q", id)
		}

		err := bkt.DeleteBucket([]byte(id))
		if err == bolt.ErrBucketNotFound {
			return errors.Wrapf(errdefs.ErrNotFound, "experiment %q", id)
		}
		return err
	})
}

func readExperiment(bkt *bolt.Bucket, experiment *Experiment) error {
	err := ReadTimestamps(bkt, &experiment.CreatedAt, &experiment.UpdatedAt)
	if err != nil {
		return err
	}

	experiment.Definition, err = readExperimentDefinition(bkt)
	if err != nil {
		return err
	}

	return bkt.ForEach(func(k, v []byte) error {
		if v == nil {
			return nil
		}

		switch string(k) {
		case string(bucketKeyID):
			experiment.ID = string(v)
		}

		return nil
	})
}

func readExperimentDefinition(bkt *bolt.Bucket) (ExperimentDefinition, error) {
	var edef ExperimentDefinition

	// dbkt := bkt.Bucket(bucketKeyDefinition)
	// if dbkt == nil {
	// 	return edef, nil
	// }

	return edef, nil
}

func writeExperiment(bkt *bolt.Bucket, experiment *Experiment) error {
	err := WriteTimestamps(bkt, experiment.CreatedAt, experiment.UpdatedAt)
	if err != nil {
		return err
	}

	err = writeExperimentDefinition(bkt, experiment.Definition)
	if err != nil {
		return err
	}

	for _, f := range []field{
		{bucketKeyID, []byte(experiment.ID)},
	} {
		err = bkt.Put(f.key, f.value)
		if err != nil {
			return err
		}
	}

	return nil
}

func writeExperimentDefinition(bkt *bolt.Bucket, edef ExperimentDefinition) error {
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

	return nil
}
