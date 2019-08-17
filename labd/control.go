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
	"fmt"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/pkg/httputil"
)

type control struct {
	addr   string
	client *httputil.Client
}

func NewControl(client *httputil.Client, addr string) p2plab.ControlAPI {
	return &control{
		addr:   addr,
		client: client,
	}
}

type urlFunc func(endpoint string, a ...interface{}) string

func (c *control) url(endpoint string, a ...interface{}) string {
	return fmt.Sprintf("%s/api/v0%s", c.addr, fmt.Sprintf(endpoint, a...))
}

func (c *control) Cluster() p2plab.ClusterAPI {
	return &clusterAPI{c.client, c.url}
}

func (c *control) Node() p2plab.NodeAPI {
	return &nodeAPI{c.client, c.url}
}

func (c *control) Scenario() p2plab.ScenarioAPI {
	return &scenarioAPI{c.client, c.url}
}

func (c *control) Benchmark() p2plab.BenchmarkAPI {
	return &benchmarkAPI{c.client, c.url}
}

func (c *control) Experiment() p2plab.ExperimentAPI {
	return &experimentAPI{c.client, c.url}
}
