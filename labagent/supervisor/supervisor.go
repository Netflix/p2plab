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

package supervisor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Netflix/p2plab/pkg/httputil"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type Supervisor interface {
	Supervise(ctx context.Context, url string) error
}

type supervisor struct {
	root    string
	appRoot string
	appPort string
	client  *httputil.Client
	app     *exec.Cmd
	cancel  func()
}

func New(root, appRoot, appAddr string, client *httputil.Client) (Supervisor, error) {
	u, err := url.Parse(appAddr)
	if err != nil {
		return nil, err
	}

	_, appPort, err := net.SplitHostPort(u.Host)
	if err != nil {
		return nil, err
	}

	return &supervisor{
		root:    root,
		appRoot: appRoot,
		appPort: appPort,
		client:  client,
	}, nil
}

func (s *supervisor) Supervise(ctx context.Context, url string) error {
	err := s.kill(ctx)
	if err != nil {
		return err
	}

	if url != "" {
		err = s.atomicReplaceBinary(ctx, url)
		if err != nil {
			return err
		}
	}

	err = s.clear(ctx)
	if err != nil {
		return err
	}

	return s.start(ctx)
}

func (s *supervisor) start(ctx context.Context) error {
	var actx context.Context
	actx, s.cancel = context.WithCancel(context.Background())
	s.app = s.cmd(actx,
		fmt.Sprintf("--root=%s", s.appRoot),
		fmt.Sprintf("--address=:%s", s.appPort),
	)
	err := s.app.Start()
	if err != nil {
		return err
	}

	v := new(bytes.Buffer)
	versionCmd := s.cmdWithStdio(ctx, v, ioutil.Discard, "--version")
	err = versionCmd.Run()
	if err != nil {
		return err
	}

	zerolog.Ctx(ctx).Debug().Str("version", v.String()).Msg("Started p2p app")
	return nil
}

func (s *supervisor) kill(ctx context.Context) error {
	if s.cancel == nil {
		return nil
	}

	s.cancel()
	defer func() {
		s.cancel = nil
		s.app = nil
	}()

	err := s.app.Wait()
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			return errors.Wrap(err, "failed to wait for app to exit")
		}

		if exitErr.ProcessState.String() != "signal: killed" {
			return errors.Wrapf(err, "app exited with unexpected status: %q", exitErr.ProcessState)
		}
	}

	zerolog.Ctx(ctx).Debug().Msg("Successfully killed app")
	return nil
}

func (s *supervisor) clear(ctx context.Context) error {
	err := os.RemoveAll(s.appRoot)
	if err != nil {
		return err
	}

	err = os.MkdirAll(s.appRoot, 0711)
	if err != nil {
		return err
	}

	return nil
}

func (s *supervisor) atomicReplaceBinary(ctx context.Context, url string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "updating binary")
	defer span.Finish()
	span.SetTag("url", url)

	zerolog.Ctx(ctx).Info().Msg("Atomically replacing binary")

	req := s.client.NewRequest("GET", url)
	resp, err := req.Send(ctx)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f, err := ioutil.TempFile(filepath.Join(s.root, "tmp"), "labapp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return err
	}

	// Atomically replace the binary.
	binaryPath := filepath.Join(s.root, "labapp")
	err = os.Rename(f.Name(), binaryPath)
	if err != nil {
		return err
	}

	return nil
}

func (s *supervisor) cmd(ctx context.Context, args ...string) *exec.Cmd {
	return s.cmdWithStdio(ctx, os.Stdout, os.Stderr, args...)
}

func (s *supervisor) cmdWithStdio(ctx context.Context, stdout, stderr io.Writer, args ...string) *exec.Cmd {
	binaryPath := filepath.Join(s.root, "labapp")
	app := exec.CommandContext(ctx, binaryPath, args...)
	app.Stdout = stdout
	app.Stderr = stderr
	return app
}
