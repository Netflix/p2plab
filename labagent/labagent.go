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

	"github.com/Netflix/p2plab/daemon"
	"github.com/Netflix/p2plab/labagent/agentrouter"
	"github.com/Netflix/p2plab/labagent/supervisor"
	"github.com/Netflix/p2plab/pkg/httputil"
	cleanhttp "github.com/hashicorp/go-cleanhttp"
	"github.com/rs/zerolog"
)

type LabAgent struct {
	daemon *daemon.Daemon
}

func New(root, addr, appRoot, appAddr string, logger *zerolog.Logger) (*LabAgent, error) {
	client, err := httputil.NewClient(cleanhttp.DefaultClient(), httputil.WithLogger(logger))
	if err != nil {
		return nil, err
	}

	s, err := supervisor.New(root, appRoot, appAddr, client)
	if err != nil {
		return nil, err
	}

	daemon := daemon.New(addr, logger,
		agentrouter.New(appAddr, client, s),
	)

	return &LabAgent{
		daemon: daemon,
	}, nil
}

func (a *LabAgent) Serve(ctx context.Context) error {
	return a.daemon.Serve(ctx)
}
