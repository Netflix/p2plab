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
	"time"

	"github.com/Netflix/p2plab/errdefs"
	metrics "github.com/libp2p/go-libp2p-core/metrics"
	peer "github.com/libp2p/go-libp2p-peer"
	protocol "github.com/libp2p/go-libp2p-protocol"
	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
)

type Report struct {
	Summary ReportSummary

	Aggregates ReportAggregates

	Nodes map[string]ReportNode
}

type ReportSummary struct {
	TotalTime time.Duration

	Trace string

	Metrics string
}

type ReportAggregates struct {
	Totals ReportNode
}

type ReportNode struct {
	Bitswap ReportBitswap

	Bandwidth ReportBandwidth
}

type ReportBitswap struct {
	BlocksReceived   uint64
	DataReceived     uint64
	BlocksSent       uint64
	DataSent         uint64
	DupBlksReceived  uint64
	DupDataReceived  uint64
	MessagesReceived uint64
}

type ReportBandwidth struct {
	Totals metrics.Stats

	// TODO: Convert back to map[node id].
	Peers map[peer.ID]metrics.Stats

	Protocols map[protocol.ID]metrics.Stats
}

func (m *db) GetReport(ctx context.Context, id string) (Report, error) {
	var report Report

	err := m.View(ctx, func(tx *bolt.Tx) error {
		bkt := getBenchmarksBucket(tx)
		if bkt == nil {
			return errors.Wrapf(errdefs.ErrNotFound, "benchmark %q", id)
		}

		bbkt := bkt.Bucket([]byte(id))
		if bbkt == nil {
			return errors.Wrapf(errdefs.ErrNotFound, "benchmark %q", id)
		}

		err := readReport(bbkt, &report)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return report, err
	}

	return report, nil
}

func (m *db) CreateReport(ctx context.Context, id string, report Report) error {
	err := m.Update(ctx, func(tx *bolt.Tx) error {
		bkt, err := createBenchmarksBucket(tx)
		if err != nil {
			return err
		}

		bbkt := bkt.Bucket([]byte(id))
		if bbkt == nil {
			return errors.Wrapf(errdefs.ErrNotFound, "benchmark %q", id)
		}

		return writeReport(bbkt, report)
	})
	if err != nil {
		return err
	}

	return nil
}

func readReport(bkt *bolt.Bucket, report *Report) error {
	content := bkt.Get(bucketKeyReport)
	if content == nil {
		return errors.Wrapf(errdefs.ErrNotFound, "no report available")
	}

	err := json.Unmarshal(content, report)
	if err != nil {
		return err
	}

	return nil
}

func writeReport(bkt *bolt.Bucket, report Report) error {
	content, err := json.Marshal(&report)
	if err != nil {
		return err
	}

	err = bkt.Put(bucketKeyReport, content)
	if err != nil {
		return err
	}

	return nil
}
