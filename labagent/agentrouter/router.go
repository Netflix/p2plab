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

package agentrouter

import (
	"context"
	"net/http"
	"time"

	"github.com/Netflix/p2plab/daemon"
	"github.com/Netflix/p2plab/labagent/supervisor"
	"github.com/Netflix/p2plab/pkg/logutil"
	"github.com/rs/zerolog"
)

type router struct {
	addr       string
	supervisor supervisor.Supervisor
}

func New(addr string, s supervisor.Supervisor) daemon.Router {
	return &router{addr, s}
}

func (s *router) Routes() []daemon.Route {
	return []daemon.Route{
		// PUT
		daemon.NewPutRoute("/update", s.putUpdate),
	}
}

func (s *router) putUpdate(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	link := r.FormValue("link")
	ctx, logger := logutil.WithResponseLogger(ctx, w)
	logger.UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.Str("link", link)
	})

	err := s.supervisor.Supervise(ctx, link)
	if err != nil {
		return err
	}

	// Give supervised process time to accept network connections.
	time.Sleep(time.Second)

	return nil
}
