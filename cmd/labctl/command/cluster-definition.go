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
	"github.com/urfave/cli"
)

var clusterDefinitionCommand = cli.Command{
	Name:    "definition",
	Aliases: []string{"d"},
	Usage:   "Manage cluster definitions.",
	Subcommands: []cli.Command{
		{
			Name:    "create",
			Aliases: []string{"c"},
			Usage:   "Creates a new cluster.",
			Action:  createClusterDefinitionAction,
		},
		{
			Name:    "remove",
			Aliases: []string{"rm"},
			Usage:   "Remove clusters.",
			Action:  removeClusterDefinitionAction,
		},
		{
			Name:    "list",
			Aliases: []string{"ls"},
			Usage:   "List clusters.",
			Action:  listClusterDefinitionAction,
		},
	},
}

func createClusterDefinitionAction(c *cli.Context) error {
	// if c.NArg() != 1 {
	// 	return errors.New("cluster definition id must be provided")
	// }

	// cln, err := ResolveClient(c)
	// if err != nil {
	// 	return err
	// }

	// cdef, err := cln.ClusterDefinition().Create(CommandContext(c), c.Args().First())
	// if err != nil {
	// 	return err
	// }

	// log.Info().Msgf("Created cluster definition %q", cdef.ID)
	return nil
}

func removeClusterDefinitionAction(c *cli.Context) error {
	// if c.NArg() != 1 {
	// 	return errors.New("cluster definition id must be provided")
	// }

	// cln, err := ResolveClient(c)
	// if err != nil {
	// 	return err
	// }

	// ctx := CommandContext(c)
	// cluster, err := cln.Cluster().Get(ctx, c.Args().First())
	// if err != nil {
	// 	return err
	// }

	// err = cluster.Remove(ctx)
	// if err != nil {
	// 	return err
	// }

	// log.Info().Msgf("Removed cluster %q", cluster.Metadata().ID)
	return nil
}

func listClusterDefinitionAction(c *cli.Context) error {
	// cln, err := ResolveClient(c)
	// if err != nil {
	// 	return err
	// }

	// clusters, err := cln.Cluster().List(CommandContext(c))
	// if err != nil {
	// 	return err
	// }

	// l := make([]interface{}, len(clusters))
	// for i, c := range clusters {
	// 	l[i] = c.Metadata()
	// }

	// return CommandPrinter(c).Print(l)
	return nil
}
