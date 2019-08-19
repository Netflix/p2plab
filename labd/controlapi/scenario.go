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
	"bytes"
	"context"
	"encoding/json"
	"strings"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/pkg/httputil"
	"github.com/pkg/errors"
)

type scenarioAPI struct {
	client *httputil.Client
	url    urlFunc
}

func (a *scenarioAPI) Create(ctx context.Context, name string, sdef metadata.ScenarioDefinition) (p2plab.Scenario, error) {
	content, err := json.MarshalIndent(&sdef, "", "    ")
	if err != nil {
		return nil, err
	}

	req := a.client.NewRequest("POST", a.url("/scenarios/create")).
		Option("name", name).
		Body(bytes.NewReader(content))

	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	s := scenario{client: a.client}
	err = json.NewDecoder(resp.Body).Decode(&s.metadata)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

func (a *scenarioAPI) Get(ctx context.Context, name string) (p2plab.Scenario, error) {
	req := a.client.NewRequest("GET", a.url("/scenarios/%s/json", name))
	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	s := scenario{client: a.client}
	err = json.NewDecoder(resp.Body).Decode(&s.metadata)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

func (a *scenarioAPI) Label(ctx context.Context, names, adds, removes []string) ([]p2plab.Scenario, error) {
	req := a.client.NewRequest("PUT", a.url("/scenarios/label")).
		Option("names", strings.Join(names, ","))

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

	var metadatas []metadata.Scenario
	err = json.NewDecoder(resp.Body).Decode(&metadatas)
	if err != nil {
		return nil, err
	}

	var scenarios []p2plab.Scenario
	for _, m := range metadatas {
		scenarios = append(scenarios, &scenario{
			client:   a.client,
			metadata: m,
		})
	}

	return scenarios, nil
}

func (a *scenarioAPI) List(ctx context.Context, opts ...p2plab.ListOption) ([]p2plab.Scenario, error) {
	var settings p2plab.ListSettings
	for _, opt := range opts {
		err := opt(&settings)
		if err != nil {
			return nil, err
		}
	}

	req := a.client.NewRequest("GET", a.url("/scenarios/json"))
	if settings.Query != "" {
		req.Option("query", settings.Query)
	}

	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var metadatas []metadata.Scenario
	err = json.NewDecoder(resp.Body).Decode(&metadatas)
	if err != nil {
		return nil, err
	}

	var scenarios []p2plab.Scenario
	for _, m := range metadatas {
		scenarios = append(scenarios, &scenario{
			client:   a.client,
			metadata: m,
		})
	}

	return scenarios, nil
}

func (a *scenarioAPI) Remove(ctx context.Context, names ...string) error {
	req := a.client.NewRequest("DELETE", a.url("/scenarios/delete")).
		Option("names", strings.Join(names, ","))

	resp, err := req.Send(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to remove scenarios")
	}
	defer resp.Body.Close()

	return nil
}

type scenario struct {
	client   *httputil.Client
	metadata metadata.Scenario
}

func (s *scenario) ID() string {
	return s.metadata.ID
}

func (s *scenario) Labels() []string {
	return s.metadata.Labels
}

func (s *scenario) Metadata() metadata.Scenario {
	return s.metadata
}
