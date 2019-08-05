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

package labapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/pkg/httputil"
)

type Client struct {
	httpClient *http.Client
	base       string
}

func NewClient(addr string) *Client {
	return &Client{
		httpClient: &http.Client{
			Transport: &http.Transport{
				Proxy:             http.ProxyFromEnvironment,
				DisableKeepAlives: true,
			},
		},
		base: fmt.Sprintf("%s/api/v0", addr),
	}
}

func (c *Client) Run(ctx context.Context, task p2plab.Task) (TaskResponse, error) {
	var taskResp TaskResponse
	content, err := json.MarshalIndent(&task, "", "    ")
	if err != nil {
		return taskResp, err
	}

	req := c.NewRequest("POST", "/run").
		Body(bytes.NewReader(content))

	resp, err := req.Send(ctx)
	if err != nil {
		return taskResp, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&taskResp)
	if err != nil {
		return taskResp, err
	}

	return taskResp, nil
}

func (c *Client) NewRequest(method, path string, a ...interface{}) *httputil.Request {
	return httputil.NewRequest(c.httpClient, c.base, method, path, a...)
}
