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
	"encoding/json"
	"strconv"
	"time"

	"github.com/Netflix/p2plab/errdefs"
	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
)

type Experiment struct {
	ID string

	Status ExperimentStatus

	Definition ExperimentDefinition

	Reports []Report

	Labels []string

	CreatedAt, UpdatedAt time.Time
}

// ToJSON is a helper function to convert an Experiment
// into it's JSON representation
func (e *Experiment) ToJSON() ([]byte, error) {
	return json.MarshalIndent(e, "", "    ")
}

// FromJSON loads the experiment definition with the values from data
func (e *Experiment) FromJSON(data []byte) error {
	return json.Unmarshal(data, e)
}

type ExperimentStatus string

var (
	ExperimentRunning ExperimentStatus = "running"
	ExperimentDone    ExperimentStatus = "done"
	ExperimentError   ExperimentStatus = "error"
)

// ExperimentDefinition defines an experiment.
type ExperimentDefinition struct {
	Trials []TrialDefinition
}

// ToJSON is a helper function to convert an ExperimentDefinition
// into it's JSON representation
func (ed *ExperimentDefinition) ToJSON() ([]byte, error) {
	return json.MarshalIndent(ed, "", "    ")
}

// FromJSON loads the experiment definition with the values from data
func (ed *ExperimentDefinition) FromJSON(data []byte) error {
	return json.Unmarshal(data, ed)
}

func (m *db) GetExperiment(ctx context.Context, id string) (Experiment, error) {
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

func (m *db) ListExperiments(ctx context.Context) ([]Experiment, error) {
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

func (m *db) CreateExperiment(ctx context.Context, experiment Experiment) (Experiment, error) {
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

func (m *db) UpdateExperiment(ctx context.Context, experiment Experiment) (Experiment, error) {
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

func (m *db) LabelExperiments(ctx context.Context, ids, adds, removes []string) ([]Experiment, error) {
	var experiments []Experiment
	err := m.Update(ctx, func(tx *bolt.Tx) error {
		bkt, err := createExperimentsBucket(tx)
		if err != nil {
			return err
		}

		err = batchUpdateLabels(bkt, ids, adds, removes, func(ibkt *bolt.Bucket, id string, labels []string) error {
			var experiment Experiment
			experiment.ID = id
			err = readExperiment(ibkt, &experiment)
			if err != nil {
				return err
			}

			experiment.Labels = labels
			experiment.UpdatedAt = time.Now().UTC()

			err = writeExperiment(ibkt, &experiment)
			if err != nil {
				return err
			}
			experiments = append(experiments, experiment)
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

	return experiments, nil
}

func (m *db) DeleteExperiment(ctx context.Context, id string) error {
	return m.Update(ctx, func(tx *bolt.Tx) error {
		bkt := getExperimentsBucket(tx)
		if bkt == nil {
			return nil
		}

		err := bkt.DeleteBucket([]byte(id))
		if err != nil {
			if err == bolt.ErrBucketNotFound {
				return errors.Wrapf(errdefs.ErrNotFound, "experiment %q", id)
			}
			return err
		}

		return nil
	})
}

func readExperiment(bkt *bolt.Bucket, experiment *Experiment) error {
	err := ReadTimestamps(bkt, &experiment.CreatedAt, &experiment.UpdatedAt)
	if err != nil {
		return err
	}

	experiment.Definition, err = readExperimentDefinition(bkt, experiment)
	if err != nil {
		return err
	}

	experiment.Labels, err = readLabels(bkt)
	if err != nil {
		return err
	}

	i := 0
	rbkt := bkt.Bucket([]byte(strconv.Itoa(i)))
	for rbkt != nil {
		var report Report
		err = readReport(rbkt, &report)
		if err != nil {
			return err
		}

		experiment.Reports = append(experiment.Reports, report)

		i++
		rbkt = bkt.Bucket([]byte(strconv.Itoa(i)))
	}

	return bkt.ForEach(func(k, v []byte) error {
		if v == nil {
			return nil
		}

		switch string(k) {
		case string(bucketKeyID):
			experiment.ID = string(v)
		case string(bucketKeyStatus):
			experiment.Status = ExperimentStatus(v)
		}

		return nil
	})
}

func readExperimentDefinition(bkt *bolt.Bucket, experiment *Experiment) (ExperimentDefinition, error) {
	var edef ExperimentDefinition
	dbkt := bkt.Bucket(bucketKeyDefinition)
	if dbkt == nil {
		return edef, nil
	}
	edefData := dbkt.Get([]byte(experiment.ID))
	if edefData == nil {
		return edef, nil
	}
	return edef, edef.FromJSON(edefData)
}

func writeExperiment(bkt *bolt.Bucket, experiment *Experiment) error {
	err := WriteTimestamps(bkt, experiment.CreatedAt, experiment.UpdatedAt)
	if err != nil {
		return err
	}

	err = writeExperimentDefinition(bkt, experiment)
	if err != nil {
		return err
	}

	err = writeLabels(bkt, experiment.Labels)
	if err != nil {
		return err
	}

	for i, report := range experiment.Reports {
		rbkt, err := bkt.CreateBucket([]byte(strconv.Itoa(i)))
		if err != nil {
			return err
		}

		err = writeReport(rbkt, report)
		if err != nil {
			return err
		}
	}

	for _, f := range []field{
		{bucketKeyID, []byte(experiment.ID)},
		{bucketKeyStatus, []byte(experiment.Status)},
	} {
		err = bkt.Put(f.key, f.value)
		if err != nil {
			return err
		}
	}

	return nil
}

func writeExperimentDefinition(bkt *bolt.Bucket, experiment *Experiment) error {
	dbkt, err := RecreateBucket(bkt, bucketKeyDefinition)
	if err != nil {
		return err
	}
	edefData, err := experiment.Definition.ToJSON()
	if err != nil {
		return err
	}
	return dbkt.Put([]byte(experiment.ID), edefData)
}
