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

package labd

import (
	"context"
	"encoding/json"
	"io"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/metadata"
)

type benchmarkAPI struct {
	cln *client
}

func (bapi *benchmarkAPI) Start(ctx context.Context, cluster, scenario string, opts ...p2plab.StartBenchmarkOption) (p2plab.Benchmark, error) {
	var settings p2plab.StartBenchmarkSettings
	for _, opt := range opts {
		err := opt(&settings)
		if err != nil {
			return nil, err
		}
	}

	req := bapi.cln.NewRequest("POST", "/benchmarks").
		Option("cluster", cluster).
		Option("scenario", scenario)

	if settings.NoReset {
		req.Option("no-reset", "true")
	}

	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b := benchmark{cln: bapi.cln}
	err = json.NewDecoder(resp.Body).Decode(&b.metadata)
	if err != nil {
		return nil, err
	}

	return &b, nil
}

func (bapi *benchmarkAPI) Get(ctx context.Context, id string) (p2plab.Benchmark, error) {
	req := bapi.cln.NewRequest("GET", "/benchmarks/%s", id)
	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b := benchmark{cln: bapi.cln}
	err = json.NewDecoder(resp.Body).Decode(&b.metadata)
	if err != nil {
		return nil, err
	}

	return &b, nil
}

func (bapi *benchmarkAPI) List(ctx context.Context) ([]p2plab.Benchmark, error) {
	req := bapi.cln.NewRequest("GET", "/benchmarks")
	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var metadatas []metadata.Benchmark
	err = json.NewDecoder(resp.Body).Decode(&metadatas)
	if err != nil {
		return nil, err
	}

	var benchmarks []p2plab.Benchmark
	for _, m := range metadatas {
		benchmarks = append(benchmarks, &benchmark{cln: bapi.cln, metadata: m})
	}

	return benchmarks, nil
}

type benchmark struct {
	cln      *client
	metadata metadata.Benchmark
}

func (b *benchmark) Metadata() metadata.Benchmark {
	return b.metadata
}

func (b *benchmark) Cancel(ctx context.Context) error {
	req := b.cln.NewRequest("GET", "/benchmarks/%s/cancel", b.metadata.ID)
	resp, err := req.Send(ctx)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (b *benchmark) Report(ctx context.Context) (p2plab.Report, error) {
	req := b.cln.NewRequest("GET", "/benchmarks/%s/report", b.metadata.ID)
	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return nil, nil
}

func (b *benchmark) Logs(ctx context.Context, opt ...p2plab.LogsOption) (io.ReadCloser, error) {
	req := b.cln.NewRequest("GET", "/benchmarks/%s/logs", b.metadata.ID)
	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	// defer resp.Body.Close()

	return resp.Body, nil
}
