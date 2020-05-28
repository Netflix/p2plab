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
	"github.com/Netflix/p2plab/pkg/logutil"
	"github.com/pkg/errors"
)

type experimentAPI struct {
	client *httputil.Client
	url    urlFunc
}

func (a *experimentAPI) Create(ctx context.Context, name string, edef metadata.ExperimentDefinition) (id string, err error) {
	content, err := edef.ToJSON()
	if err != nil {
		return id, err
	}

	req := a.client.NewRequest("POST", a.url("/experiments/create"), httputil.WithRetryMax(0)).
		Option("id", name).
		Body(bytes.NewReader(content))

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

func (a *experimentAPI) Get(ctx context.Context, id string) (p2plab.Experiment, error) {
	req := a.client.NewRequest("GET", a.url("/experiments/%s/json", id))
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

func (a *experimentAPI) Label(ctx context.Context, ids, adds, removes []string) ([]p2plab.Experiment, error) {
	req := a.client.NewRequest("PUT", a.url("/experiments/label")).
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
		})
	}

	return experiments, nil
}
func (a *experimentAPI) List(ctx context.Context, opts ...p2plab.ListOption) ([]p2plab.Experiment, error) {
	var settings p2plab.ListSettings
	for _, opt := range opts {
		err := opt(&settings)
		if err != nil {
			return nil, err
		}
	}

	req := a.client.NewRequest("GET", a.url("/experiments/json"))
	if settings.Query != "" {
		req.Option("query", settings.Query)
	}

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
		})
	}

	return experiments, nil
}

func (a *experimentAPI) Remove(ctx context.Context, ids ...string) error {
	req := a.client.NewRequest("DELETE", a.url("/experiments/delete")).
		Option("ids", strings.Join(ids, ","))

	resp, err := req.Send(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to remove experiments")
	}
	defer resp.Body.Close()

	return nil
}

type experiment struct {
	client   *httputil.Client
	metadata metadata.Experiment
}

func (e *experiment) ID() string {
	return e.metadata.ID
}

func (e *experiment) Labels() []string {
	return e.metadata.Labels
}

func (e *experiment) Metadata() metadata.Experiment {
	return e.metadata
}
