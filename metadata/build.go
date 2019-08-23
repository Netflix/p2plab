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

type Build struct {
	ID string

	Link string

	CreatedAt, UpdatedAt time.Time
}

func (m *db) GetBuild(ctx context.Context, id string) (Build, error) {
	var build Build

	err := m.View(ctx, func(tx *bolt.Tx) error {
		bkt := getBuildsBucket(tx)
		if bkt == nil {
			return errors.Wrapf(errdefs.ErrNotFound, "build %q", id)
		}

		cbkt := bkt.Bucket([]byte(id))
		if cbkt == nil {
			return errors.Wrapf(errdefs.ErrNotFound, "build %q", id)
		}

		build.ID = id
		err := readBuild(cbkt, &build)
		if err != nil {
			return errors.Wrapf(err, "build %q", id)
		}

		return nil
	})
	if err != nil {
		return Build{}, err
	}

	return build, nil
}

func (m *db) ListBuilds(ctx context.Context) ([]Build, error) {
	var builds []Build
	err := m.View(ctx, func(tx *bolt.Tx) error {
		bkt := getBuildsBucket(tx)
		if bkt == nil {
			return nil
		}

		return bkt.ForEach(func(k, v []byte) error {
			var (
				build = Build{
					ID: string(k),
				}
				cbkt = bkt.Bucket(k)
			)

			err := readBuild(cbkt, &build)
			if err != nil {
				return err
			}

			builds = append(builds, build)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	return builds, nil
}

func (m *db) CreateBuild(ctx context.Context, build Build) (Build, error) {
	err := m.Update(ctx, func(tx *bolt.Tx) error {
		bkt, err := createBuildsBucket(tx)
		if err != nil {
			return err
		}

		cbkt, err := bkt.CreateBucket([]byte(build.ID))
		if err != nil {
			if err != bolt.ErrBucketExists {
				return err
			}

			return errors.Wrapf(errdefs.ErrAlreadyExists, "build %q", build.ID)
		}

		build.CreatedAt = time.Now().UTC()
		build.UpdatedAt = build.CreatedAt
		return writeBuild(cbkt, &build)
	})
	if err != nil {
		return Build{}, err
	}
	return build, err
}

func (m *db) DeleteBuild(ctx context.Context, id string) error {
	return m.Update(ctx, func(tx *bolt.Tx) error {
		bkt := getBuildsBucket(tx)
		if bkt == nil {
			return nil
		}

		err := bkt.DeleteBucket([]byte(id))
		if err != nil {
			if err == bolt.ErrBucketNotFound {
				return errors.Wrapf(errdefs.ErrNotFound, "build %q", id)
			}
			return err
		}

		return nil
	})
}

func readBuild(bkt *bolt.Bucket, build *Build) error {
	err := ReadTimestamps(bkt, &build.CreatedAt, &build.UpdatedAt)
	if err != nil {
		return err
	}

	return bkt.ForEach(func(k, v []byte) error {
		if v == nil {
			return nil
		}

		switch string(k) {
		case string(bucketKeyID):
			build.ID = string(v)
		case string(bucketKeyLink):
			build.Link = string(v)
		}

		return nil
	})
}

func writeBuild(bkt *bolt.Bucket, build *Build) error {
	err := WriteTimestamps(bkt, build.CreatedAt, build.UpdatedAt)
	if err != nil {
		return err
	}

	for _, f := range []field{
		{bucketKeyID, []byte(build.ID)},
		{bucketKeyLink, []byte(build.Link)},
	} {
		err = bkt.Put(f.key, f.value)
		if err != nil {
			return err
		}
	}

	return nil
}
