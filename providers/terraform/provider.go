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

package terraform

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"text/template"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/errdefs"
	"github.com/Netflix/p2plab/metadata"
	"github.com/pkg/errors"
)

type provider struct {
	root          string
	tfvars        *template.Template
	maintf        *template.Template
	terraformById map[string]*Terraform
}

type BackendVars struct {
	Bucket string
	Key    string
	Region string
}

type ClusterVars struct {
	ID                    string
	ClusterGroupsByRegion []RegionalClusterGroup
}

type RegionalClusterGroup struct {
	Region string
	Groups []metadata.ClusterGroup
}

func New(root string) (p2plab.PeerProvider, error) {
	tfvarsPath := filepath.Join(root, "terraform/terraform.tfvars")
	tfvars, err := template.New("terraform.tfvars").Parse(tfvarsPath)
	if err != nil {
		return nil, err
	}

	maintfPath := filepath.Join(root, "terraform/terraform.maintf")
	maintf, err := template.New("main.tf").Parse(maintfPath)
	if err != nil {
		return nil, err
	}

	return &provider{
		root:          root,
		tfvars:        tfvars,
		maintf:        maintf,
		terraformById: make(map[string]*Terraform),
	}, nil
}

func (p *provider) CreatePeerGroup(ctx context.Context, id string, cdef metadata.ClusterDefinition) (*p2plab.PeerGroup, error) {
	clusterDir, err := p.prepareClusterDir(id)
	if err != nil {
		return nil, err
	}

	err = p.executeTfvarsTemplate(id, cdef)
	if err != nil {
		return nil, err
	}

	t, err := NewTerraform(ctx, clusterDir)
	if err != nil {
		return nil, err
	}
	p.terraformById[id] = t

	// addrs, err := t.Apply(ctx)
	// if err != nil {
	// 	return nil, err
	// }

	pg := &p2plab.PeerGroup{ID: id}
	// for _, addr := range addrs {
	// 	pg.Peers = append(pg.Peers, labagent.NewClient(addr))
	// }

	return pg, nil
}

func (p *provider) DestroyPeerGroup(ctx context.Context, pg *p2plab.PeerGroup) error {
	t, ok := p.terraformById[pg.ID]
	if !ok {
		return errors.Wrapf(errdefs.ErrNotFound, "terraform not found for %q", pg.ID)
	}

	err := t.Destroy(ctx)
	if err != nil {
		return err
	}
	defer t.Close()

	delete(p.terraformById, pg.ID)
	return p.destroyClusterDir(pg.ID)
}

func (p *provider) prepareClusterDir(id string) (clusterDir string, err error) {
	clusterDir = filepath.Join(p.root, id, "terraform")
	_, err = os.Stat(clusterDir)
	if err == nil {
		return clusterDir, errors.Errorf("cluster terraform dir already exists: %q", clusterDir)
	}

	moduleDir := filepath.Join(clusterDir, "modules/labagent")
	err = os.MkdirAll(moduleDir, 0775)
	if err != nil {
		return clusterDir, err
	}
	defer func() {
		if err != nil {
			os.Remove(clusterDir)
		}
	}()

	terraformDir := filepath.Join(p.root, "terraform")
	for _, p := range []string{
		"outputs.tf",
		"variables.tf",
		"modules/labagent/main.tf",
		"modules/labagent/outputs.tf",
		"modules/labagent/variables.tf",
	} {
		err = os.Symlink(filepath.Join(terraformDir, p), filepath.Join(clusterDir, p))
		if err != nil {
			return clusterDir, err
		}
	}

	maintfPath := filepath.Join(p.root, id, "terraform/main.tf")
	f, err := os.Create(maintfPath)
	if err != nil {
		return clusterDir, err
	}
	defer f.Close()

	vars := BackendVars{
		Bucket: "nflx-labdterraform-protocollabstest-us-west-2",
		Key:    id,
		Region: "us-west-2",
	}

	err = p.maintf.Execute(f, &vars)
	if err != nil {
		return clusterDir, err
	}

	return clusterDir, nil
}

func (p *provider) destroyClusterDir(id string) error {
	clusterDir := filepath.Join(p.root, id, "terraform")
	return os.RemoveAll(clusterDir)
}

func (p *provider) executeTfvarsTemplate(id string, cdef metadata.ClusterDefinition) error {
	vars := ClusterVars{ID: id}

	clusterGroupsByRegion := make(map[string]*RegionalClusterGroup)
	for _, group := range cdef.Groups {
		rcg, ok := clusterGroupsByRegion[group.Region]
		if !ok {
			rcg = &RegionalClusterGroup{
				Region: group.Region,
			}
		}

		rcg.Groups = append(rcg.Groups, group)
		clusterGroupsByRegion[group.Region] = rcg
	}

	for _, rcg := range clusterGroupsByRegion {
		vars.ClusterGroupsByRegion = append(vars.ClusterGroupsByRegion, *rcg)
	}
	sort.SliceStable(vars.ClusterGroupsByRegion, func(i, j int) bool {
		return vars.ClusterGroupsByRegion[i].Region < vars.ClusterGroupsByRegion[j].Region
	})

	tfvarsPath := filepath.Join(p.root, id, "terraform/terraform.tfvars")
	f, err := os.Create(tfvarsPath)
	if err != nil {
		return err
	}
	defer f.Close()

	err = p.tfvars.Execute(f, &vars)
	if err != nil {
		return err
	}

	return nil
}
