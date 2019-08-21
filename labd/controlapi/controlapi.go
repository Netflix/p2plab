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
	"fmt"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/pkg/httputil"
)

const (
	ResourceID = "ResourceID"
)

type api struct {
	addr   string
	client *httputil.Client
}

func New(client *httputil.Client, addr string) p2plab.ControlAPI {
	return &api{
		addr:   addr,
		client: client,
	}
}

type urlFunc func(endpoint string, v ...interface{}) string

func (a *api) url(endpoint string, v ...interface{}) string {
	return fmt.Sprintf("%s%s", a.addr, fmt.Sprintf(endpoint, v...))
}

func (a *api) Cluster() p2plab.ClusterAPI {
	return &clusterAPI{a.client, a.url}
}

func (a *api) Node() p2plab.NodeAPI {
	return &nodeAPI{a.client, a.url}
}

func (a *api) Scenario() p2plab.ScenarioAPI {
	return &scenarioAPI{a.client, a.url}
}

func (a *api) Benchmark() p2plab.BenchmarkAPI {
	return &benchmarkAPI{a.client, a.url}
}

func (a *api) Experiment() p2plab.ExperimentAPI {
	return &experimentAPI{a.client, a.url}
}
