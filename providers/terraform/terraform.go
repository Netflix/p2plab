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
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"sort"

	"github.com/Netflix/p2plab/errdefs"
	"github.com/Netflix/p2plab/metadata"
	"github.com/pkg/errors"
)

type Terraform struct {
	root    string
	leaseCh chan struct{}
}

type Output struct {
	InstancesByRegion map[string]Instances
}

type Instances struct {
	IDs        []string `json:"ids"`
	PrivateIPs []string `json:"private_ips"`
}

func NewTerraform(ctx context.Context, root string) (*Terraform, error) {
	leaseCh := make(chan struct{}, 1)
	leaseCh <- struct{}{}

	t := &Terraform{
		root:    root,
		leaseCh: leaseCh,
	}

	err := t.terraform(ctx, "init")
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (t *Terraform) Apply(ctx context.Context) ([]metadata.Node, error) {
	err := t.acquireLease()
	if err != nil {
		return nil, err
	}
	defer func() {
		t.leaseCh <- struct{}{}
	}()

	err = t.terraform(ctx, "apply", "-auto-approve")
	if err != nil {
		return nil, errors.Wrap(err, "failed to auto-approve apply templates")
	}

	stdout := new(bytes.Buffer)
	err = t.terraformWithStdio(ctx, stdout, ioutil.Discard, "output", "-json")
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute json output")
	}

	var output Output
	err = json.NewDecoder(stdout).Decode(&output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode json output")
	}

	var regions []string
	for region := range output.InstancesByRegion {
		regions = append(regions, region)
	}
	sort.Strings(regions)

	var nodes []metadata.Node
	for _, region := range regions {
		instances := output.InstancesByRegion[region]

		if len(instances.IDs) != len(instances.PrivateIPs) {
			return nil, errors.Errorf("number of instance ids (%d) don't match number of private ips (%d)", len(instances.IDs), len(instances.PrivateIPs))
		}

		for i := 0; i < len(instances.IDs); i++ {
			nodes = append(nodes, metadata.Node{
				ID:      instances.IDs[i],
				Address: instances.PrivateIPs[i],
			})
		}
	}

	return nodes, nil
}

func (t *Terraform) Destroy(ctx context.Context) error {
	err := t.acquireLease()
	if err != nil {
		return err
	}
	defer func() {
		t.leaseCh <- struct{}{}
	}()

	return t.terraform(ctx, "destroy", "-auto-approve")
}

func (t *Terraform) acquireLease() error {
	select {
	case _, ok := <-t.leaseCh:
		if !ok {
			return errors.Wrapf(errdefs.ErrUnavailable, "terraform handler leases chan already closed")
		}
		return nil
	default:
		return errors.Wrapf(errdefs.ErrUnavailable, "terraform operation already in progress")
	}
}

func (t *Terraform) Close() {
	close(t.leaseCh)
}

func (t *Terraform) terraform(ctx context.Context, args ...string) error {
	return t.terraformWithStdio(ctx, os.Stdout, os.Stderr, args...)
}

func (t *Terraform) terraformWithStdio(ctx context.Context, stdout, stderr io.Writer, args ...string) error {
	cmd := exec.CommandContext(ctx, "terraform", args...)
	cmd.Dir = t.root
	cmd.Stdin = nil
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
