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
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"text/template"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/errdefs"
	"github.com/Netflix/p2plab/metadata"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
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
	RegionalClusterGroups []RegionalClusterGroups
}

type RegionalClusterGroups struct {
	Region string
	Groups []metadata.ClusterGroup
}

func New(root string) (p2plab.NodeProvider, error) {
	tfvarsPath := filepath.Join(root, "templates/terraform.tfvars")
	tfvarsContent, err := ioutil.ReadFile(tfvarsPath)
	if err != nil {
		return nil, err
	}

	tfvars, err := template.New("terraform.tfvars").Parse(string(tfvarsContent))
	if err != nil {
		return nil, err
	}

	maintfPath := filepath.Join(root, "templates/main.tf")
	maintfContent, err := ioutil.ReadFile(maintfPath)
	if err != nil {
		return nil, err
	}

	maintf, err := template.New("main.tf").Parse(string(maintfContent))
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

func (p *provider) CreateNodeGroup(ctx context.Context, id string, cdef metadata.ClusterDefinition) (*p2plab.NodeGroup, error) {
	log.Debug().Str("id", id).Msg("Preparing cluster directory")
	clusterDir, err := p.prepareClusterDir(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare cluster directory")
	}

	log.Debug().Str("id", id).Str("dir", clusterDir).Msg("Executing tfvars template")
	err = p.executeTfvarsTemplate(id, cdef)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute tfvars template")
	}

	log.Debug().Str("id", id).Str("dir", clusterDir).Msg("Creating terraform handler")
	t, err := NewTerraform(ctx, clusterDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create terraform handler")
	}
	p.terraformById[id] = t

	log.Debug().Str("id", id).Str("dir", clusterDir).Msg("Terraform applying")
	ns, err := t.Apply(ctx, id, cdef)
	if err != nil {
		return nil, errors.Wrap(err, "failed to terraform destroy")
	}

	return &p2plab.NodeGroup{
		ID:    id,
		Nodes: ns,
	}, nil
}

func (p *provider) DestroyNodeGroup(ctx context.Context, ng *p2plab.NodeGroup) error {
	t, ok := p.terraformById[ng.ID]
	if !ok {
		log.Debug().Str("id", ng.ID).Msg("Creating terraform handler")
		var err error
		clusterDir := filepath.Join(p.root, ng.ID)
		t, err = NewTerraform(ctx, clusterDir)
		if err != nil {
			return errors.Wrap(err, "failed to create terraform handler")
		}
		p.terraformById[ng.ID] = t
	}

	log.Debug().Str("id", ng.ID).Msg("Terraform destroying")
	err := t.Destroy(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to terraform destroy")
	}
	defer t.Close()

	delete(p.terraformById, ng.ID)
	log.Debug().Str("id", ng.ID).Msg("Removing cluster directory")
	err = p.destroyClusterDir(ng.ID)
	if err != nil {
		return errors.Wrap(err, "failed to remove cluster directory")
	}

	return nil
}

func (p *provider) prepareClusterDir(id string) (clusterDir string, err error) {
	clusterDir = filepath.Join(p.root, id)
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

	terraformDir := filepath.Join(p.root, "templates")
	for _, p := range []string{
		"outputs.tf",
		"variables.tf",
		"modules/labagent/main.tf",
		"modules/labagent/outputs.tf",
		"modules/labagent/variables.tf",
	} {
		dst, err := filepath.Abs(filepath.Join(terraformDir, p))
		if err != nil {
			return clusterDir, err
		}

		src, err := filepath.Abs(filepath.Join(clusterDir, p))
		if err != nil {
			return clusterDir, err
		}

		err = os.Symlink(dst, src)
		if err != nil {
			return clusterDir, err
		}
	}

	maintfPath := filepath.Join(p.root, id, "main.tf")
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
	clusterDir := filepath.Join(p.root, id)
	return os.RemoveAll(clusterDir)
}

func (p *provider) executeTfvarsTemplate(id string, cdef metadata.ClusterDefinition) error {
	vars := ClusterVars{ID: id}

	clusterGroupsByRegion := map[string]RegionalClusterGroups{
		"us-west-2": RegionalClusterGroups{Region: "us-west-2"},
		"us-east-1": RegionalClusterGroups{Region: "us-east-1"},
		"eu-west-1": RegionalClusterGroups{Region: "eu-west-1"},
	}

	for _, group := range cdef.Groups {
		rcg, ok := clusterGroupsByRegion[group.Region]
		if !ok {
			return errors.Wrapf(errdefs.ErrInvalidArgument, "unsupported region %q", group.Region)
		}

		rcg.Groups = append(rcg.Groups, group)
		clusterGroupsByRegion[group.Region] = rcg
	}

	for _, rcg := range clusterGroupsByRegion {
		vars.RegionalClusterGroups = append(vars.RegionalClusterGroups, rcg)
	}
	sort.SliceStable(vars.RegionalClusterGroups, func(i, j int) bool {
		return vars.RegionalClusterGroups[i].Region < vars.RegionalClusterGroups[j].Region
	})

	tfvarsPath := filepath.Join(p.root, id, "terraform.tfvars")
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
