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
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/Netflix/p2plab/errdefs"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/pkg/logutil"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type Terraform struct {
	root    string
	leaseCh chan struct{}
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

func (t *Terraform) Apply(ctx context.Context, id string, cdef metadata.ClusterDefinition) ([]metadata.Node, error) {
	err := t.acquireLease()
	if err != nil {
		return nil, err
	}
	defer func() {
		t.leaseCh <- struct{}{}
	}()

	span, ctx := opentracing.StartSpanFromContext(ctx, "terraform apply")
	defer span.Finish()
	span.SetTag("cluster", id)

	if zerolog.Ctx(ctx).GetLevel() == zerolog.InfoLevel {
		ectx, cancel := context.WithCancel(ctx)
		defer cancel()

		go logutil.Elapsed(ectx, 20*time.Second, func(ctx context.Context, elapsed time.Duration) {
			zerolog.Ctx(ctx).Info().Dur("elapsed", elapsed).Msg("Applying terraform configuration")
		})
	}

	err = t.terraform(ctx, "apply", "-auto-approve")
	if err != nil {
		return nil, errors.Wrap(err, "failed to auto-approve apply templates")
	}

	var ns []metadata.Node
	for i, cg := range cdef.Groups {
		asg := fmt.Sprintf("%s-%d", id, i)

		zerolog.Ctx(ctx).Debug().Str("asg", asg).Msg("Discovering instances in ASG")
		instances, err := DiscoverInstances(ctx, asg, cg.Region)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to discover instances for ASG %q in %q", asg, cg.Region)
		}

		for _, instance := range instances {
			ns = append(ns, metadata.Node{
				ID:      instance.InstanceId,
				Address: instance.PrivateIp,
				Labels: append([]string{
					instance.InstanceId,
					instance.InstanceType,
					cg.Region,
				}, cg.Labels...),
			})
		}
	}

	return ns, nil
}

func (t *Terraform) Destroy(ctx context.Context, id string) error {
	err := t.acquireLease()
	if err != nil {
		return err
	}
	defer func() {
		t.leaseCh <- struct{}{}
	}()

	span, ctx := opentracing.StartSpanFromContext(ctx, "terraform destroy")
	defer span.Finish()
	span.SetTag("cluster", id)

	if zerolog.Ctx(ctx).GetLevel() == zerolog.InfoLevel {
		ectx, cancel := context.WithCancel(ctx)
		defer cancel()

		go logutil.Elapsed(ectx, 20*time.Second, func(ctx context.Context, elapsed time.Duration) {
			zerolog.Ctx(ctx).Info().Dur("elapsed", elapsed).Msg("Applying terraform configuration")
		})
	}

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
	logger := zerolog.Ctx(ctx).With().Strs("exec", args).Logger()
	logWriter := logutil.NewWriter(&logger, zerolog.DebugLevel)
	defer logWriter.Close()

	return t.terraformWithStdio(ctx, logWriter, logWriter, args...)
}

func (t *Terraform) terraformWithStdio(ctx context.Context, stdout, stderr io.Writer, args ...string) error {
	cmd := exec.CommandContext(ctx, "terraform", args...)
	cmd.Dir = t.root
	cmd.Stdin = nil
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
