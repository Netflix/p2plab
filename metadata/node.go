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
	"sort"
	"time"

	"github.com/Netflix/p2plab/errdefs"
	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
)

type Node struct {
	ID string

	Labels []string

	CreatedAt, UpdatedAt time.Time
}

func (m *DB) GetNode(ctx context.Context, cluster, id string) (Node, error) {
	var node Node

	err := m.View(func(tx *bolt.Tx) error {
		bkt := getNodesBucket(tx, cluster)
		if bkt == nil {
			return errors.Wrapf(errdefs.ErrNotFound, "node %q", id)
		}

		cbkt := bkt.Bucket([]byte(id))
		if cbkt == nil {
			return errors.Wrapf(errdefs.ErrNotFound, "node %q", id)
		}

		node.ID = id
		err := readNode(cbkt, &node)
		if err != nil {
			return errors.Wrapf(err, "node %q", id)
		}

		return nil
	})
	if err != nil {
		return Node{}, err
	}

	return node, nil
}

func (m *DB) ListNodes(ctx context.Context, cluster string) ([]Node, error) {
	var nodes []Node
	err := m.View(func(tx *bolt.Tx) error {
		bkt := getNodesBucket(tx, cluster)
		if bkt == nil {
			return nil
		}

		return bkt.ForEach(func(k, v []byte) error {
			var (
				node = Node{
					ID: string(k),
				}
				cbkt = bkt.Bucket(k)
			)

			err := readNode(cbkt, &node)
			if err != nil {
				return err
			}

			nodes = append(nodes, node)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	return nodes, nil
}

func (m *DB) CreateNode(ctx context.Context, cluster string, node Node) (Node, error) {
	err := m.Update(func(tx *bolt.Tx) error {
		bkt, err := createNodesBucket(tx, cluster)
		if err != nil {
			return err
		}

		cbkt, err := bkt.CreateBucket([]byte(node.ID))
		if err != nil {
			if err != bolt.ErrBucketExists {
				return err
			}

			return errors.Wrapf(errdefs.ErrAlreadyExists, "node %q", node.ID)
		}

		node.CreatedAt = time.Now().UTC()
		node.UpdatedAt = node.CreatedAt
		return writeNode(cbkt, &node)
	})
	if err != nil {
		return Node{}, err
	}
	return node, err
}

func (m *DB) LabelNodes(ctx context.Context, cluster string, ids, addLabels, removeLabels []string) ([]Node, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	addSet := make(map[string]struct{})
	for _, l := range addLabels {
		addSet[l] = struct{}{}
	}

	removeSet := make(map[string]struct{})
	for _, l := range removeLabels {
		removeSet[l] = struct{}{}
	}

	var nodes []Node
	err := m.Update(func(tx *bolt.Tx) error {
		bkt, err := createNodesBucket(tx, cluster)
		if err != nil {
			return err
		}

		for _, id := range ids {
			cbkt := bkt.Bucket([]byte(id))
			if cbkt == nil {
				return errors.Wrapf(errdefs.ErrNotFound, "node %q", id)
			}

			lbkt := cbkt.Bucket(bucketKeyLabels)

			var labels []string
			if lbkt != nil {
				err = lbkt.ForEach(func(k, v []byte) error {
					if _, ok := removeSet[string(k)]; ok {
						return nil
					}

					labels = append(labels, string(k))
					delete(addSet, string(k))
					return nil
				})
				if err != nil {
					return err
				}

				err = cbkt.DeleteBucket(bucketKeyLabels)
				if err != nil {
					return err
				}
			}

			for l, _ := range addSet {
				labels = append(labels, l)
			}
			sort.Strings(labels)

			lbkt, err = cbkt.CreateBucket(bucketKeyLabels)
			if err != nil {
				return err
			}

			var node Node
			node.ID = id
			err = readNode(cbkt, &node)
			if err != nil {
				return err
			}

			node.Labels = labels
			node.UpdatedAt = time.Now().UTC()

			err = writeNode(cbkt, &node)
			if err != nil {
				return err
			}
			nodes = append(nodes, node)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return nodes, nil
}

func (m *DB) DeleteNode(ctx context.Context, cluster, id string) error {
	return m.Update(func(tx *bolt.Tx) error {
		bkt := getNodesBucket(tx, cluster)
		if bkt == nil {
			return errors.Wrapf(errdefs.ErrNotFound, "node %q", id)
		}

		err := bkt.DeleteBucket([]byte(id))
		if err == bolt.ErrBucketNotFound {
			return errors.Wrapf(errdefs.ErrNotFound, "node %q", id)
		}
		return err
	})
}

func readNode(bkt *bolt.Bucket, node *Node) error {
	err := ReadTimestamps(bkt, &node.CreatedAt, &node.UpdatedAt)
	if err != nil {
		return err
	}

	return bkt.ForEach(func(k, v []byte) error {
		if v == nil {
			return nil
		}

		switch string(k) {
		// case string(bucketKeyField):
		//  node.Field = string(v)
		}

		return nil
	})
}

func writeNode(bkt *bolt.Bucket, node *Node) error {
	err := WriteTimestamps(bkt, node.CreatedAt, node.UpdatedAt)
	if err != nil {
		return err
	}

	for _, f := range []field{
		// {bucketKeyField, []byte(node.Field)},
	} {
		err = bkt.Put(f.key, f.value)
		if err != nil {
			return err
		}
	}

	if len(node.Labels) > 0 {
		lbkt, err := bkt.CreateBucketIfNotExists(bucketKeyLabels)
		if err != nil {
			return err
		}

		for _, l := range node.Labels {
			err = lbkt.Put([]byte(l), nil)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
