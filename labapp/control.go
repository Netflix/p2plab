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
	"time"

	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/pkg/httputil"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	"github.com/rs/zerolog/log"
)

type Control struct {
	addr   string
	client *httputil.Client
}

func NewControl(client *httputil.Client, addr string) *Control {
	return &Control{
		addr:   addr,
		client: client,
	}
}

func (c *Control) url(endpoint string, a ...interface{}) string {
	return fmt.Sprintf("%s/api/v0%s", c.addr, fmt.Sprintf(endpoint, a...))
}

func (c *Control) Healthcheck(ctx context.Context) bool {
	req := c.client.NewRequest("GET", c.url("/healthcheck"),
		httputil.WithRetryWaitMax(time.Minute),
		httputil.WithRetryMax(10),
	)
	resp, err := req.Send(ctx)
	if err != nil {
		log.Debug().Str("err", err.Error()).Str("addr", c.addr).Msg("unhealthy")
		return false
	}
	defer resp.Body.Close()

	return true
}

func (c *Control) PeerInfo(ctx context.Context) (peerstore.PeerInfo, error) {
	req := c.client.NewRequest("GET", c.url("/peerInfo"))

	var peerInfo peerstore.PeerInfo
	resp, err := req.Send(ctx)
	if err != nil {
		return peerInfo, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&peerInfo)
	if err != nil {
		return peerInfo, err
	}

	return peerInfo, nil
}

func (c *Control) Run(ctx context.Context, task metadata.Task) error {
	content, err := json.MarshalIndent(&task, "", "    ")
	if err != nil {
		return err
	}

	req := c.client.NewRequest("POST", c.url("/run")).
		Body(bytes.NewReader(content))

	resp, err := req.Send(ctx)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
