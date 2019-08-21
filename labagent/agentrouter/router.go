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
	"github.com/Netflix/p2plab/errdefs"
	"github.com/Netflix/p2plab/labagent/supervisor"
	"github.com/Netflix/p2plab/labapp/appapi"
	"github.com/Netflix/p2plab/pkg/httputil"
	"github.com/Netflix/p2plab/pkg/logutil"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type router struct {
	addr       string
	client     *httputil.Client
	supervisor supervisor.Supervisor
}

func New(addr string, client *httputil.Client, s supervisor.Supervisor) daemon.Router {
	return &router{addr, client, s}
}

func (s *router) Routes() []daemon.Route {
	return []daemon.Route{
		// PUT
		daemon.NewPutRoute("/update", s.putUpdate),
	}
}

func (s *router) putUpdate(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	url := r.FormValue("url")
	ctx, logger := logutil.WithResponseLogger(ctx, w)
	logger.UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.Str("url", url)
	})

	err := s.supervisor.Supervise(ctx, url)
	if err != nil {
		return err
	}

	// Give supervised process time to accept network connections.
	time.Sleep(time.Second)

	app := appapi.New(s.client, s.addr)
	healthy := app.Healthcheck(ctx)
	if !healthy {
		return errors.Wrap(errdefs.ErrUnavailable, "labapp unhealthy")
	}

	return nil
}
