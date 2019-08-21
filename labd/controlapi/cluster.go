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
	"os"
	"strings"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/pkg/httputil"
	"github.com/Netflix/p2plab/pkg/logutil"
	"github.com/pkg/errors"
)

type clusterAPI struct {
	client *httputil.Client
	url    urlFunc
}

func (a *clusterAPI) Create(ctx context.Context, name string, opts ...p2plab.CreateClusterOption) (id string, err error) {
	var settings p2plab.CreateClusterSettings
	for _, opt := range opts {
		err := opt(&settings)
		if err != nil {
			return id, err
		}
	}

	var cdef metadata.ClusterDefinition
	if settings.Definition != "" {
		f, err := os.Open(settings.Definition)
		if err != nil {
			return id, err
		}
		defer f.Close()

		err = json.NewDecoder(f).Decode(&cdef)
		if err != nil {
			return id, err
		}
	} else {
		cdef.Groups = append(cdef.Groups, metadata.ClusterGroup{
			Size:         settings.Size,
			InstanceType: settings.InstanceType,
			Region:       settings.Region,
		})
	}

	content, err := json.MarshalIndent(&cdef, "", "    ")
	if err != nil {
		return id, err
	}

	req := a.client.NewRequest("POST", a.url("/clusters/create"), httputil.WithRetryMax(0)).
		Option("name", name).
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

func (a *clusterAPI) Get(ctx context.Context, name string) (p2plab.Cluster, error) {
	req := a.client.NewRequest("GET", a.url("/clusters/%s/json", name))
	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	c := cluster{client: a.client, url: a.url}
	err = json.NewDecoder(resp.Body).Decode(&c.metadata)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (a *clusterAPI) Label(ctx context.Context, names, adds, removes []string) ([]p2plab.Cluster, error) {
	req := a.client.NewRequest("PUT", a.url("/clusters/label")).
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

	var metadatas []metadata.Cluster
	err = json.NewDecoder(resp.Body).Decode(&metadatas)
	if err != nil {
		return nil, err
	}

	var clusters []p2plab.Cluster
	for _, m := range metadatas {
		clusters = append(clusters, &cluster{
			client:   a.client,
			metadata: m,
			url:      a.url,
		})
	}

	return clusters, nil
}

func (a *clusterAPI) List(ctx context.Context, opts ...p2plab.ListOption) ([]p2plab.Cluster, error) {
	var settings p2plab.ListSettings
	for _, opt := range opts {
		err := opt(&settings)
		if err != nil {
			return nil, err
		}
	}

	req := a.client.NewRequest("GET", a.url("/clusters/json"))
	if settings.Query != "" {
		req.Option("query", settings.Query)
	}

	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var metadatas []metadata.Cluster
	err = json.NewDecoder(resp.Body).Decode(&metadatas)
	if err != nil {
		return nil, err
	}

	var clusters []p2plab.Cluster
	for _, m := range metadatas {
		clusters = append(clusters, &cluster{
			client:   a.client,
			metadata: m,
			url:      a.url,
		})
	}

	return clusters, nil
}

type Event struct {
}

func (a *clusterAPI) Remove(ctx context.Context, names ...string) error {
	req := a.client.NewRequest("DELETE", a.url("/clusters/delete")).
		Option("names", strings.Join(names, ","))

	resp, err := req.Send(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to remove clusters")
	}
	defer resp.Body.Close()

	logWriter := logutil.LogWriter(ctx)
	if logWriter != nil {
		err = logutil.WriteRemoteLogs(ctx, resp.Body, logWriter)
		if err != nil {
			return err
		}
	}

	return nil
}

type cluster struct {
	client   *httputil.Client
	metadata metadata.Cluster
	url      urlFunc
}

func (c *cluster) ID() string {
	return c.metadata.ID
}

func (c *cluster) Labels() []string {
	return c.metadata.Labels
}

func (c *cluster) Metadata() metadata.Cluster {
	return c.metadata
}

func (c *cluster) Update(ctx context.Context, commit string) error {
	req := c.client.NewRequest("PUT", c.url("/clusters/%s/update", c.metadata.ID)).
		Option("commit", commit)

	resp, err := req.Send(ctx)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	logWriter := logutil.LogWriter(ctx)
	if logWriter != nil {
		err = logutil.WriteRemoteLogs(ctx, resp.Body, logWriter)
		if err != nil {
			return err
		}
	}

	return nil
}
