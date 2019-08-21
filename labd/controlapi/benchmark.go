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

package controlapi

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/pkg/httputil"
	"github.com/Netflix/p2plab/pkg/logutil"
	"github.com/pkg/errors"
)

type benchmarkAPI struct {
	client *httputil.Client
	url    urlFunc
}

func (a *benchmarkAPI) Start(ctx context.Context, cluster, scenario string, opts ...p2plab.StartBenchmarkOption) (id string, err error) {
	var settings p2plab.StartBenchmarkSettings
	for _, opt := range opts {
		err = opt(&settings)
		if err != nil {
			return id, err
		}
	}

	req := a.client.NewRequest("POST", a.url("/benchmarks/create"), httputil.WithRetryMax(0)).
		Option("cluster", cluster).
		Option("scenario", scenario)

	if settings.NoReset {
		req.Option("no-reset", "true")
	}

	resp, err := req.Send(ctx)
	if err != nil {
		return id, err
	}
	defer resp.Body.Close()

	logWriter := logutil.LogWriter(ctx)
	if logWriter != nil {
		err = logutil.WriteRemoteLogs(ctx, resp.Body, logWriter)
		if err != nil {
			return id, err
		}
	}

	return resp.Header.Get(ResourceID), nil
}

func (a *benchmarkAPI) Get(ctx context.Context, id string) (p2plab.Benchmark, error) {
	req := a.client.NewRequest("GET", a.url("/benchmarks/%s/json", id))
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

func (a *benchmarkAPI) Label(ctx context.Context, ids, adds, removes []string) ([]p2plab.Benchmark, error) {
	req := a.client.NewRequest("PUT", a.url("/benchmarks/label")).
		Option("ids", strings.Join(ids, ","))

	if len(adds) > 0 {
		req.Option("adds", strings.Join(adds, ","))
	}
	if len(removes) > 0 {
		req.Option("removes", strings.Join(removes, ","))
	}

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
		})
	}

	return benchmarks, nil
}

func (a *benchmarkAPI) List(ctx context.Context, opts ...p2plab.ListOption) ([]p2plab.Benchmark, error) {
	var settings p2plab.ListSettings
	for _, opt := range opts {
		err := opt(&settings)
		if err != nil {
			return nil, err
		}
	}

	req := a.client.NewRequest("GET", a.url("/benchmarks/json"))
	if settings.Query != "" {
		req.Option("query", settings.Query)
	}

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
		})
	}

	return benchmarks, nil
}

func (a *benchmarkAPI) Remove(ctx context.Context, ids ...string) error {
	req := a.client.NewRequest("DELETE", a.url("/benchmarks/delete")).
		Option("ids", strings.Join(ids, ","))

	resp, err := req.Send(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to remove benchmarks")
	}
	defer resp.Body.Close()

	return nil
}

type benchmark struct {
	client   *httputil.Client
	metadata metadata.Benchmark
}

func (b *benchmark) ID() string {
	return b.metadata.ID
}

func (b *benchmark) Labels() []string {
	return b.metadata.Labels
}

func (b *benchmark) Metadata() metadata.Benchmark {
	return b.metadata
}
