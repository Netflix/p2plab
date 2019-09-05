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
	"strings"
	"time"

	"github.com/Netflix/p2plab/errdefs"
	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
)

var (
	DefaultPeerDefinition = PeerDefinition{
		GitReference:       "HEAD",
		Transports:         []string{"tcp"},
		Muxers:             []string{"mplex"},
		SecurityTransports: []string{"secio"},
		Routing:            "nil",
	}
)

type Node struct {
	ID string

	Address string

	AgentPort int

	AppPort int

	Peer PeerDefinition

	Labels []string

	CreatedAt, UpdatedAt time.Time
}

type PeerDefinition struct {
	GitReference string

	Transports []string

	Muxers []string

	SecurityTransports []string

	Routing string
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

func (m *db) UpdateNode(ctx context.Context, cluster string, node Node) (Node, error) {
	if node.ID == "" {
		return Node{}, errors.Wrapf(errdefs.ErrInvalidArgument, "node id required for update")
	}

	err := m.Update(ctx, func(tx *bolt.Tx) error {
		bkt, err := createNodesBucket(tx, cluster)
		if err != nil {
			return err
		}

		cbkt := bkt.Bucket([]byte(node.ID))
		if cbkt == nil {
			return errors.Wrapf(errdefs.ErrNotFound, "node %q", node.ID)
		}

		node.UpdatedAt = time.Now().UTC()
		return writeNode(cbkt, &node)
	})
	if err != nil {
		return Node{}, err
	}

	return node, nil
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

	node.Peer, err = readPeerDefinition(bkt)
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
		case string(bucketKeyAgentPort):
			node.AgentPort, _ = strconv.Atoi(string(v))
		case string(bucketKeyAppPort):
			node.AppPort, _ = strconv.Atoi(string(v))
		}

		return nil
	})
}

func readPeerDefinition(bkt *bolt.Bucket) (PeerDefinition, error) {
	var pdef PeerDefinition

	dbkt := bkt.Bucket(bucketKeyDefinition)
	if dbkt == nil {
		return pdef, nil
	}

	err := dbkt.ForEach(func(k, v []byte) error {
		if v == nil {
			return nil
		}

		switch string(k) {
		case string(bucketKeyGitReference):
			pdef.GitReference = string(v)
		case string(bucketKeyTransports):
			if len(v) > 0 {
				pdef.Transports = strings.Split(string(v), ",")
			}
		case string(bucketKeyMuxers):
			if len(v) > 0 {
				pdef.Muxers = strings.Split(string(v), ",")
			}
		case string(bucketKeySecurityTransports):
			if len(v) > 0 {
				pdef.SecurityTransports = strings.Split(string(v), ",")
			}
		case string(bucketKeyRouting):
			pdef.Routing = string(v)
		}

		return nil
	})
	return pdef, err
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

	err = writePeerDefinition(bkt, node.Peer)
	if err != nil {
		return err
	}

	for _, f := range []field{
		{bucketKeyID, []byte(node.ID)},
		{bucketKeyAddress, []byte(node.Address)},
		{bucketKeyAgentPort, []byte(strconv.Itoa(node.AgentPort))},
		{bucketKeyAppPort, []byte(strconv.Itoa(node.AppPort))},
	} {
		err = bkt.Put(f.key, f.value)
		if err != nil {
			return err
		}
	}

	return nil
}

func writePeerDefinition(bkt *bolt.Bucket, pdef PeerDefinition) error {
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

	for _, f := range []field{
		{bucketKeyGitReference, []byte(pdef.GitReference)},
		{bucketKeyTransports, []byte(strings.Join(pdef.Transports, ","))},
		{bucketKeyMuxers, []byte(strings.Join(pdef.Muxers, ","))},
		{bucketKeySecurityTransports, []byte(strings.Join(pdef.SecurityTransports, ","))},
		{bucketKeyRouting, []byte(pdef.Routing)},
	} {
		err = dbkt.Put(f.key, f.value)
		if err != nil {
			return err
		}
	}

	return nil
}
