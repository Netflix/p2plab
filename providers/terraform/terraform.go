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
	"io"
	"os"
	"os/exec"

	"github.com/Netflix/p2plab/errdefs"
	"github.com/pkg/errors"
)

type Terraform struct {
	root    string
	leaseCh chan struct{}
}

func NewTerraform(ctx context.Context, root string) (*Terraform, error) {
	leaseCh := make(chan struct{})
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

func (t *Terraform) Apply(ctx context.Context) ([]string, error) {
	lease, ok := <-t.leaseCh
	if !ok {
		return nil, errors.Wrapf(errdefs.ErrUnavailable, "terraform operation already in progress")
	}
	defer func() {
		t.leaseCh <- lease
	}()

	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	err := t.terraformWithStdio(ctx, stdout, stderr, "apply")
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (t *Terraform) Destroy(ctx context.Context) error {
	lease, ok := <-t.leaseCh
	if !ok {
		return errors.Wrapf(errdefs.ErrUnavailable, "terraform operation already in progress")
	}
	defer func() {
		t.leaseCh <- lease
	}()

	return t.terraform(ctx, "destroy")
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
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
