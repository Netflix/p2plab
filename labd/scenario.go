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

	"github.com/Netflix/p2plab"
)

type scenarioAPI struct {
	cln *client
}

func (sapi *scenarioAPI) Create(ctx context.Context, name string, sdef p2plab.ScenarioDefinition) (p2plab.Scenario, error) {
	req := sapi.cln.NewRequest("POST", "/scenarios")
	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return &scenario{
		cln:  sapi.cln,
		name: name,
	}, nil
}

func (sapi *scenarioAPI) Get(ctx context.Context, name string) (p2plab.Scenario, error) {
	req := sapi.cln.NewRequest("HEAD", "/scenarios/%s", name)
	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return &scenario{
		cln:  sapi.cln,
		name: name,
	}, nil
}

func (sapi *scenarioAPI) List(ctx context.Context) ([]p2plab.Scenario, error) {
	req := sapi.cln.NewRequest("GET", "/scenarios")
	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return nil, nil
}

type scenario struct {
	cln  *client
	name string
}

func (s *scenario) Remove(ctx context.Context) error {
	req := s.cln.NewRequest("DELETE", "/scenarios/%s", s.name)
	resp, err := req.Send(ctx)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
