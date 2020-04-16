// Copyright 2019 Netflix, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by apilicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package appapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/pkg/httputil"
	"github.com/Netflix/p2plab/pkg/logutil"
	"github.com/libp2p/go-libp2p-core/peer"
)

type api struct {
	addr   string
	client *httputil.Client
}

func New(client *httputil.Client, addr string) p2plab.AppAPI {
	return &api{
		addr:   addr,
		client: client,
	}
}

func (a *api) url(endpoint string, v ...interface{}) string {
	return fmt.Sprintf("%s%s", a.addr, fmt.Sprintf(endpoint, v...))
}

func (a *api) PeerInfo(ctx context.Context) (peer.AddrInfo, error) {
	var peerInfo peer.AddrInfo

	req := a.client.NewRequest("GET", a.url("/peerInfo"))
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

func (a *api) Report(ctx context.Context) (metadata.ReportNode, error) {
	var report metadata.ReportNode

	req := a.client.NewRequest("GET", a.url("/report"))
	resp, err := req.Send(ctx)
	if err != nil {
		return report, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&report)
	if err != nil {
		return report, err
	}

	return report, nil
}

func (a *api) Run(ctx context.Context, task metadata.Task) error {
	content, err := json.MarshalIndent(&task, "", "    ")
	if err != nil {
		return err
	}

	req := a.client.NewRequest("POST", a.url("/run")).
		Body(bytes.NewReader(content))

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
