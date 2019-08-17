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

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/pkg/httputil"
	"github.com/pkg/errors"
)

type experimentAPI struct {
	client *httputil.Client
	url    urlFunc
}

func (a *experimentAPI) Start(ctx context.Context, id string, edef metadata.ExperimentDefinition) (p2plab.Experiment, error) {
	content, err := json.MarshalIndent(&edef, "", "    ")
	if err != nil {
		return nil, err
	}

	req := a.client.NewRequest("POST", a.url("/experiments")).
		Option("id", id).
		Body(bytes.NewReader(content))

	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	e := experiment{client: a.client}
	err = json.NewDecoder(resp.Body).Decode(&e.metadata)
	if err != nil {
		return nil, err
	}

	return &e, nil
}

func (a *experimentAPI) Get(ctx context.Context, id string) (p2plab.Experiment, error) {
	req := a.client.NewRequest("GET", a.url("/experiments/%s", id))
	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	e := experiment{client: a.client, url: a.url}
	err = json.NewDecoder(resp.Body).Decode(&e.metadata)
	if err != nil {
		return nil, err
	}

	return &e, nil
}

func (a *experimentAPI) List(ctx context.Context) ([]p2plab.Experiment, error) {
	req := a.client.NewRequest("GET", a.url("/experiments"))
	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var metadatas []metadata.Experiment
	err = json.NewDecoder(resp.Body).Decode(&metadatas)
	if err != nil {
		return nil, err
	}

	var experiments []p2plab.Experiment
	for _, m := range metadatas {
		experiments = append(experiments, &experiment{
			client:   a.client,
			metadata: m,
			url:      a.url,
		})
	}

	return experiments, nil
}

type experiment struct {
	client   *httputil.Client
	metadata metadata.Experiment
	url      urlFunc
}

func (e *experiment) Metadata() metadata.Experiment {
	return e.metadata
}

func (e *experiment) Cancel(ctx context.Context) error {
	req := e.client.NewRequest("PUT", e.url("/experiments/%s/cancel", e.metadata.ID))
	resp, err := req.Send(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to cancel experiment %q", e.metadata.ID)
	}
	defer resp.Body.Close()

	return nil
}
