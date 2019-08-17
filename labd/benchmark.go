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
	"github.com/Netflix/p2plab/pkg/httputil"
)

type benchmarkAPI struct {
	client *httputil.Client
	url    urlFunc
}

func (a *benchmarkAPI) Start(ctx context.Context, cluster, scenario string, opts ...p2plab.StartBenchmarkOption) (p2plab.Benchmark, error) {
	var settings p2plab.StartBenchmarkSettings
	for _, opt := range opts {
		err := opt(&settings)
		if err != nil {
			return nil, err
		}
	}

	req := a.client.NewRequest("POST", a.url("/benchmarks")).
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

	b := benchmark{client: a.client}
	err = json.NewDecoder(resp.Body).Decode(&b.metadata)
	if err != nil {
		return nil, err
	}

	return &b, nil
}

func (a *benchmarkAPI) Get(ctx context.Context, id string) (p2plab.Benchmark, error) {
	req := a.client.NewRequest("GET", a.url("/benchmarks/%s", id))
	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b := benchmark{client: a.client}
	err = json.NewDecoder(resp.Body).Decode(&b.metadata)
	if err != nil {
		return nil, err
	}

	return &b, nil
}

func (a *benchmarkAPI) List(ctx context.Context) ([]p2plab.Benchmark, error) {
	req := a.client.NewRequest("GET", a.url("/benchmarks"))
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
		benchmarks = append(benchmarks, &benchmark{
			client:   a.client,
			metadata: m,
			url:      a.url,
		})
	}

	return benchmarks, nil
}

type benchmark struct {
	client   *httputil.Client
	metadata metadata.Benchmark
	url      urlFunc
}

func (b *benchmark) Metadata() metadata.Benchmark {
	return b.metadata
}

func (b *benchmark) Cancel(ctx context.Context) error {
	req := b.client.NewRequest("GET", b.url("/benchmarks/%s/cancel", b.metadata.ID))
	resp, err := req.Send(ctx)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (b *benchmark) Report(ctx context.Context) (p2plab.Report, error) {
	req := b.client.NewRequest("GET", b.url("/benchmarks/%s/report", b.metadata.ID))
	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return nil, nil
}

func (b *benchmark) Logs(ctx context.Context, opt ...p2plab.LogsOption) (io.ReadCloser, error) {
	req := b.client.NewRequest("GET", b.url("/benchmarks/%s/logs", b.metadata.ID))
	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	// defer resp.Body.Close()

	return resp.Body, nil
}
