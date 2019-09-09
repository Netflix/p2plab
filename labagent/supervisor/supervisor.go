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
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Netflix/p2plab/downloaders"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/pkg/httputil"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type Supervisor interface {
	Supervise(ctx context.Context, link string, pdef metadata.PeerDefinition) error
}

type supervisor struct {
	root    string
	appRoot string
	appPort string
	client  *httputil.Client
	fs      *downloaders.Downloaders
	app     *exec.Cmd
	cancel  func()
}

func New(root, appRoot, appAddr string, client *httputil.Client, fs *downloaders.Downloaders) (Supervisor, error) {
	err := os.MkdirAll(root, 0711)
	if err != nil {
		return nil, err
	}

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
		fs:      fs,
	}, nil
}

func (s *supervisor) Supervise(ctx context.Context, link string, pdef metadata.PeerDefinition) error {
	err := s.kill(ctx)
	if err != nil {
		return err
	}

	flags := s.peerDefinitionToFlags(pdef)
	if link != "" {
		err = s.atomicReplaceBinary(ctx, link)
		if err != nil {
			return err
		}

		err = s.clear(ctx)
		if err != nil {
			return err
		}

		return s.start(ctx, flags)
	} else {
		return s.wait(ctx, flags)
	}

}

func (s *supervisor) peerDefinitionToFlags(pdef metadata.PeerDefinition) []string {
	flags := []string{
		fmt.Sprintf("--root=%s", s.appRoot),
		fmt.Sprintf("--address=:%s", s.appPort),
	}

	for _, transportType := range pdef.Transports {
		flags = append(flags, fmt.Sprintf("--libp2p-transports=%s", transportType))
	}
	for _, muxerType := range pdef.Muxers {
		flags = append(flags, fmt.Sprintf("--libp2p-muxers=%s", muxerType))
	}
	for _, securityTransportType := range pdef.SecurityTransports {
		flags = append(flags, fmt.Sprintf("--libp2p-security-transports=%s", securityTransportType))
	}
	if pdef.Routing != "" {
		flags = append(flags, fmt.Sprintf("--libp2p-routing=%s", pdef.Routing))
	}

	return flags
}

func (s *supervisor) start(ctx context.Context, flags []string) error {
	var actx context.Context
	actx, s.cancel = context.WithCancel(context.Background())
	s.app = s.cmd(actx, flags...)
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

func (s *supervisor) wait(ctx context.Context, flags []string) error {
	span := opentracing.SpanFromContext(ctx)
	if span != nil {
		buf := new(bytes.Buffer)
		err := opentracing.GlobalTracer().Inject(
			span.Context(),
			opentracing.Binary,
			buf,
		)
		if err != nil {
			return errors.Wrap(err, "failed to inject trace into buffer")
		}

		trace := base64.StdEncoding.EncodeToString(buf.Bytes())
		flags = append(flags, fmt.Sprintf("--trace=%s", trace))
	}

	s.app = s.cmd(ctx, flags...)
	return s.app.Run()
}

func (s *supervisor) kill(ctx context.Context) error {
	defer func() {
		s.cancel = nil
		s.app = nil
	}()

	if s.cancel == nil {
		return nil
	}

	s.cancel()
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

func (s *supervisor) atomicReplaceBinary(ctx context.Context, link string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "updating binary")
	defer span.Finish()
	span.SetTag("link", link)

	zerolog.Ctx(ctx).Debug().Msg("Atomically replacing binary")
	u, err := url.Parse(link)
	if err != nil {
		return err
	}

	downloader, err := s.fs.Get(u.Scheme)
	if err != nil {
		return err
	}

	rc, err := downloader.Download(ctx, link)
	if err != nil {
		return err
	}
	defer rc.Close()

	f, err := ioutil.TempFile(s.root, "labapp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()

	_, err = io.Copy(f, rc)
	if err != nil {
		return err
	}

	err = os.Chmod(f.Name(), 0775)
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
