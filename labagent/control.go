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

package labagent

import (
	"context"
	"fmt"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/pkg/httputil"
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

func (c *Control) Update(ctx context.Context, url string) error {
	req := c.client.NewRequest("PUT", c.url("/update")).
		Option("url", url)

	resp, err := req.Send(ctx)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (c *Control) SSH(ctx context.Context, opts ...p2plab.SSHOption) error {
	return nil
}
