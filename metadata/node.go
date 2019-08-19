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

type Node struct {
	ID string

	Address string

	Labels []string

	CreatedAt, UpdatedAt time.Time
}

func (m *db) GetNode(ctx context.Context, cluster, id string) (Node, error) {
	var node Node

	err := m.View(ctx, func(tx *bolt.Tx) error {
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

func (m *db) ListNodes(ctx context.Context, cluster string) ([]Node, error) {
	var nodes []Node
	err := m.View(ctx, func(tx *bolt.Tx) error {
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

func (m *db) CreateNode(ctx context.Context, cluster string, node Node) (Node, error) {
	nodes, err := m.CreateNodes(ctx, cluster, []Node{node})
	if err != nil {
		return Node{}, err
	}

	if len(nodes) != 1 {
		return Node{}, errors.New("failed to retrieve created node")
	}

	return nodes[0], nil
}

func (m *db) CreateNodes(ctx context.Context, cluster string, nodes []Node) ([]Node, error) {
	err := m.Update(ctx, func(tx *bolt.Tx) error {
		bkt, err := createNodesBucket(tx, cluster)
		if err != nil {
			return err
		}

		for i, node := range nodes {
			cbkt, err := bkt.CreateBucket([]byte(node.ID))
			if err != nil {
				if err != bolt.ErrBucketExists {
					return err
				}

				return errors.Wrapf(errdefs.ErrAlreadyExists, "node %q", node.ID)
			}

			node.CreatedAt = time.Now().UTC()
			node.UpdatedAt = node.CreatedAt
			err = writeNode(cbkt, &node)
			if err != nil {
				return err
			}

			nodes[i] = node
		}

		return nil

	})
	if err != nil {
		return nil, err
	}
	return nodes, err
}

func (m *db) LabelNodes(ctx context.Context, cluster string, ids, adds, removes []string) ([]Node, error) {
	var nodes []Node
	err := m.Update(ctx, func(tx *bolt.Tx) error {
		bkt, err := createNodesBucket(tx, cluster)
		if err != nil {
			return err
		}

		err = batchUpdateLabels(bkt, ids, adds, removes, func(ibkt *bolt.Bucket, id string, labels []string) error {
			var node Node
			node.ID = id
			err = readNode(ibkt, &node)
			if err != nil {
				return err
			}

			node.Labels = labels
			node.UpdatedAt = time.Now().UTC()

			err = writeNode(ibkt, &node)
			if err != nil {
				return err
			}
			nodes = append(nodes, node)
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

	return nodes, nil
}

func (m *db) DeleteNodes(ctx context.Context, cluster string, ids ...string) error {
	return m.Update(ctx, func(tx *bolt.Tx) error {
		bkt := getNodesBucket(tx, cluster)
		if bkt == nil {
			return nil
		}

		for _, id := range ids {
			err := bkt.DeleteBucket([]byte(id))
			if err != nil {
				if err == bolt.ErrBucketNotFound {
					return errors.Wrapf(errdefs.ErrNotFound, "node %q", id)
				}
				return err
			}

		}

		return nil
	})
}

func readNode(bkt *bolt.Bucket, node *Node) error {
	err := ReadTimestamps(bkt, &node.CreatedAt, &node.UpdatedAt)
	if err != nil {
		return err
	}

	node.Labels, err = readLabels(bkt)
	if err != nil {
		return err
	}

	return bkt.ForEach(func(k, v []byte) error {
		if v == nil {
			return nil
		}

		switch string(k) {
		case string(bucketKeyID):
			node.ID = string(v)
		case string(bucketKeyAddress):
			node.Address = string(v)
		}

		return nil
	})
}

func writeNode(bkt *bolt.Bucket, node *Node) error {
	err := WriteTimestamps(bkt, node.CreatedAt, node.UpdatedAt)
	if err != nil {
		return err
	}

	err = writeLabels(bkt, node.Labels)
	if err != nil {
		return err
	}

	for _, f := range []field{
		{bucketKeyID, []byte(node.ID)},
		{bucketKeyAddress, []byte(node.Address)},
	} {
		err = bkt.Put(f.key, f.value)
		if err != nil {
			return err
		}
	}

	return nil
}
