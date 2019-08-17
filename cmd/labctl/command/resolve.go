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

package command

import (
	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/labagent/agentapi"
	"github.com/Netflix/p2plab/labapp/appapi"
	"github.com/Netflix/p2plab/labd/controlapi"
	"github.com/urfave/cli"
)

func ResolveControl(c *cli.Context) (p2plab.ControlAPI, error) {
	api := controlapi.New(CommandClient(c), c.GlobalString("address"))
	// TODO: healthcheck
	return api, nil
}

func ResolveAgent(c *cli.Context, addr string) (p2plab.AgentAPI, error) {
	api := agentapi.New(CommandClient(c), addr)
	// TODO: healthcheck
	return api, nil
}

func ResolveApplication(c *cli.Context, addr string) (p2plab.AppAPI, error) {
	api := appapi.New(CommandClient(c), addr)
	// TODO: healthcheck
	return api, nil
}
