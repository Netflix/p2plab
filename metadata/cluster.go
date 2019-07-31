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

type Cluster struct {
	ID string

	CreatedAt, UpdatedAt time.Time
}

func (m *DB) GetCluster(ctx context.Context, id string) (Cluster, error) {
	var cluster Cluster

	err := m.View(func(tx *bolt.Tx) error {
		bkt := getClustersBucket(tx)
		if bkt == nil {
			return errors.Wrapf(errdefs.ErrNotFound, "cluster %q", id)
		}

		cbkt := bkt.Bucket([]byte(id))
		if cbkt == nil {
			return errors.Wrapf(errdefs.ErrNotFound, "cluster %q", id)
		}

		cluster.ID = id
		err := readCluster(cbkt, &cluster)
		if err != nil {
			return errors.Wrapf(err, "cluster %q", id)
		}

		return nil
	})
	if err != nil {
		return Cluster{}, err
	}

	return cluster, nil
}

func (m *DB) ListClusters(ctx context.Context) ([]Cluster, error) {
	var clusters []Cluster
	err := m.View(func(tx *bolt.Tx) error {
		bkt := getClustersBucket(tx)
		if bkt == nil {
			return nil
		}

		return bkt.ForEach(func(k, v []byte) error {
			var (
				cluster = Cluster{
					ID: string(k),
				}
				cbkt = bkt.Bucket(k)
			)

			err := readCluster(cbkt, &cluster)
			if err != nil {
				return err
			}

			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	return clusters, nil
}

func (m *DB) CreateCluster(ctx context.Context, cluster Cluster) (Cluster, error) {
	err := m.Update(func(tx *bolt.Tx) error {
		bkt, err := createClustersBucket(tx)
		if err != nil {
			return err
		}

		cbkt, err := bkt.CreateBucket([]byte(cluster.ID))
		if err != nil {
			if err != bolt.ErrBucketExists {
				return err
			}

			return errors.Wrapf(errdefs.ErrAlreadyExists, "cluster %q", cluster.ID)
		}

		cluster.CreatedAt = time.Now().UTC()
		cluster.UpdatedAt = cluster.CreatedAt
		return writeCluster(cbkt, &cluster)
	})
	if err != nil {
		return Cluster{}, err
	}
	return cluster, err
}

func (m *DB) UpdateCluster(ctx context.Context, cluster Cluster) (Cluster, error) {
	if cluster.ID == "" {
		return Cluster{}, errors.Wrapf(errdefs.ErrInvalidArgument, "cluster id required for update")
	}

	err := m.Update(func(tx *bolt.Tx) error {
		bkt, err := createClustersBucket(tx)
		if err != nil {
			return err
		}

		cbkt := bkt.Bucket([]byte(cluster.ID))
		if cbkt == nil {
			return errors.Wrapf(errdefs.ErrNotFound, "cluster %q", cluster.ID)
		}

		cluster.UpdatedAt = time.Now().UTC()
		return writeCluster(cbkt, &cluster)
	})
	if err != nil {
		return Cluster{}, err
	}

	return cluster, nil
}

func (m *DB) DeleteCluster(ctx context.Context, id string) error {
	return m.Update(func(tx *bolt.Tx) error {
		bkt := getClustersBucket(tx)
		if bkt == nil {
			return errors.Wrapf(errdefs.ErrNotFound, "cluster %q", id)
		}

		err := bkt.DeleteBucket([]byte(id))
		if err == bolt.ErrBucketNotFound {
			return errors.Wrapf(errdefs.ErrNotFound, "cluster %q", id)
		}
		return err
	})
}

func readCluster(bkt *bolt.Bucket, cluster *Cluster) error {
	err := ReadTimestamps(bkt, &cluster.CreatedAt, &cluster.UpdatedAt)
	if err != nil {
		return err
	}

	return bkt.ForEach(func(k, v []byte) error {
		if v == nil {
			return nil
		}

		switch string(k) {
		// case string(bucketKeyField):
		//  cluster.Field = string(v)
		}

		return nil
	})
}

func writeCluster(bkt *bolt.Bucket, cluster *Cluster) error {
	err := WriteTimestamps(bkt, cluster.CreatedAt, cluster.UpdatedAt)
	if err != nil {
		return err
	}

	for _, f := range []field{
		// {bucketKeyField, []byte(cluster.Field)},
	} {
		err = bkt.Put(f.key, f.value)
		if err != nil {
			return err
		}
	}

	return nil
}
