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
	"bytes"
	"context"
	"encoding/json"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/metadata"
	"github.com/pkg/errors"
)

type experimentAPI struct {
	cln *client
}

func (eapi *experimentAPI) Start(ctx context.Context, id string, edef metadata.ExperimentDefinition) (p2plab.Experiment, error) {
	content, err := json.MarshalIndent(&edef, "", "    ")
	if err != nil {
		return nil, err
	}

	req := eapi.cln.NewRequest("POST", "/experiments").
		Option("id", id).
		Body(bytes.NewReader(content))

	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	e := experiment{cln: eapi.cln}
	err = json.NewDecoder(resp.Body).Decode(&e.metadata)
	if err != nil {
		return nil, err
	}

	return &e, nil
}

func (eapi *experimentAPI) Get(ctx context.Context, id string) (p2plab.Experiment, error) {
	req := eapi.cln.NewRequest("GET", "/experiments/%s", id)
	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	e := experiment{cln: eapi.cln}
	err = json.NewDecoder(resp.Body).Decode(&e.metadata)
	if err != nil {
		return nil, err
	}

	return &e, nil
}

func (eapi *experimentAPI) List(ctx context.Context) ([]p2plab.Experiment, error) {
	req := eapi.cln.NewRequest("GET", "/experiments")
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
		experiments = append(experiments, &experiment{cln: eapi.cln, metadata: m})
	}

	return experiments, nil
}

type experiment struct {
	cln      *client
	metadata metadata.Experiment
}

func (e *experiment) Metadata() metadata.Experiment {
	return e.metadata
}

func (e *experiment) Cancel(ctx context.Context) error {
	req := e.cln.NewRequest("PUT", "/experiments/%s/cancel", e.metadata.ID)
	resp, err := req.Send(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to cancel experiment %q", e.metadata.ID)
	}
	defer resp.Body.Close()

	return nil
}
