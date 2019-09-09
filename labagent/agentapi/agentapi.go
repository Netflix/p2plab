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

package agentapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/pkg/httputil"
	"github.com/Netflix/p2plab/pkg/logutil"
	"github.com/rs/zerolog"
)

type api struct {
	addr   string
	client *httputil.Client
}

func New(client *httputil.Client, addr string) p2plab.AgentAPI {
	return &api{
		addr:   addr,
		client: client,
	}
}

func (a *api) url(endpoint string, v ...interface{}) string {
	return fmt.Sprintf("%s%s", a.addr, fmt.Sprintf(endpoint, v...))
}

func (a *api) Healthcheck(ctx context.Context) bool {
	req := a.client.NewRequest("GET", a.url("/healthcheck"),
		httputil.WithRetryWaitMax(5*time.Minute),
		httputil.WithRetryMax(10),
	)
	resp, err := req.Send(ctx)
	if err != nil {
		zerolog.Ctx(ctx).Debug().Str("err", err.Error()).Str("addr", a.addr).Msg("unhealthy")
		return false
	}
	defer resp.Body.Close()

	return true
}

func (a *api) Update(ctx context.Context, id, link string, pdef metadata.PeerDefinition) error {
	content, err := json.MarshalIndent(&pdef, "", "    ")
	if err != nil {
		return err
	}

	req := a.client.NewRequest("PUT", a.url("/update")).
		Body(bytes.NewReader(content)).
		Option("id", id).
		Option("link", link)

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

func (a *api) SSH(ctx context.Context, opts ...p2plab.SSHOption) error {
	return nil
}
