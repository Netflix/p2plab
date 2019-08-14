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

	"github.com/Netflix/p2plab/metadata"
)

type clusterDefinitionAPI struct {
	cln *client
}

func (cdapi *clusterDefinitionAPI) Create(ctx context.Context, id string, cdef metadata.ClusterDefinition) error {
	content, err := json.MarshalIndent(&cdef, "", "    ")
	if err != nil {
		return err
	}

	req := cdapi.cln.NewRequest("POST", "/clusterDefinitions").
		Option("id", id).
		Body(bytes.NewReader(content))

	resp, err := req.Send(ctx)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (cdapi *clusterDefinitionAPI) Remove(ctx context.Context, id string) error {
	req := cdapi.cln.NewRequest("DELETE", "/clusterDefinitions/%s", id)
	resp, err := req.Send(ctx)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (cdapi *clusterDefinitionAPI) List(ctx context.Context) ([]metadata.ClusterDefinition, error) {
	req := cdapi.cln.NewRequest("GET", "/clusterDefinitions")
	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var cdefs []metadata.ClusterDefinition
	err = json.NewDecoder(resp.Body).Decode(&cdefs)
	if err != nil {
		return nil, err
	}

	return cdefs, nil
}
