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

package labagent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/Netflix/p2plab/errdefs"
	"github.com/Netflix/p2plab/labapp"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/pkg/httputil"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

var (
	LabAppBinary = "labapp"
	LabAppFifo   = "labapp-fifo"
)

type LabAgent struct {
	root       string
	addr       string
	appRoot    string
	appAddr    string
	router     *mux.Router
	httpClient *http.Client
	app        *exec.Cmd
	appClient  *labapp.Client
	appCancel  func()
}

func New(root, addr, appRoot, appAddr string) (*LabAgent, error) {
	r := mux.NewRouter().UseEncodedPath().StrictSlash(true)
	agent := &LabAgent{
		root:    root,
		addr:    addr,
		appRoot: appRoot,
		appAddr: appAddr,
		router:  r,
		httpClient: &http.Client{
			Transport: &http.Transport{
				Proxy:             http.ProxyFromEnvironment,
				DisableKeepAlives: true,
			},
		},
		appClient: labapp.NewClient("http://localhost:7003"),
	}
	agent.registerRoutes(r)

	return agent, nil
}

func (a *LabAgent) Serve(ctx context.Context) error {
	log.Info().Msgf("labagent listening on %s", a.addr)
	s := &http.Server{
		Handler:      a.router,
		Addr:         a.addr,
		ReadTimeout:  10 * time.Second,
	}

	// TODO: remove when S3 update flow is complete
	err := a.updateApp(ctx, "")
	if err != nil {
		return err
	}

	return s.ListenAndServe()
}

func (a *LabAgent) registerRoutes(r *mux.Router) {
	api := r.PathPrefix("/api/v0").Subrouter()
	api.Handle("/peerInfo", httputil.ErrorHandler{a.runHandler}).Methods("GET")
	api.Handle("/run", httputil.ErrorHandler{a.runHandler}).Methods("POST")
}

func (a *LabAgent) peerInfoHandler(w http.ResponseWriter, r *http.Request) error {
	peerInfo, err := a.appClient.PeerInfo(r.Context())
	if err != nil {
		return err
	}
	return httputil.WriteJSON(w, &peerInfo)
}

func (a *LabAgent) runHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("labagent/run")

	var task metadata.Task
	err := json.NewDecoder(r.Body).Decode(&task)
	if err != nil {
		return err
	}

	switch task.Type {
	case metadata.TaskUpdate:
		var resp labapp.TaskResponse
		err = a.updateApp(r.Context(), task.Subject)
		if err != nil {
			resp.Err = err.Error()
		}
		return httputil.WriteJSON(w, &resp)
	case metadata.TaskGet, metadata.TaskConnect:
		if a.appCancel == nil {
			return errors.Wrapf(errdefs.ErrInvalidArgument, "no labapp currently running")
		}

		resp, err := a.appClient.Run(r.Context(), task)
		if err != nil {
			return err
		}
		return httputil.WriteJSON(w, &resp)
	default:
		return errors.Wrapf(errdefs.ErrInvalidArgument, "unrecognized task type: %q", task.Type)
	}
}

func (a *LabAgent) updateApp(ctx context.Context, url string) error {
	err := a.killApp()
	if err != nil {
		return err
	}

	// err = a.updateBinary(url)
	// if err != nil {
	// 	return err
	// }

	err = a.startApp()
	if err != nil {
		return err
	}

	return nil
}

func (a *LabAgent) killApp() error {
	if a.appCancel != nil {
		// Kill subprocess.
		a.appCancel()

		err := a.app.Wait()
		if err != nil {
			exitErr, ok := err.(*exec.ExitError)
			if !ok {
				return errors.Wrap(err, "failed to wait for labapp to exit")
			}

			if exitErr.ProcessState.String() != "signal: killed" {
				return errors.Wrapf(err, "labapp exited with unexpected status: %q", exitErr.ProcessState)
			}
		}

		log.Info().Msg("Successfully killed labapp")
	}

	return nil
}

func (a *LabAgent) updateBinary(url string) error {
	resp, err := a.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f, err := ioutil.TempFile(filepath.Join(a.root, "tmp"), "labapp")
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
	binaryPath := filepath.Join(a.root, LabAppBinary)
	err = os.Rename(f.Name(), binaryPath)
	if err != nil {
		return err
	}

	return nil
}

func (a *LabAgent) startApp() error {
	var appCtx context.Context
	appCtx, a.appCancel = context.WithCancel(context.Background())

	binaryPath := filepath.Join(a.root, LabAppBinary)
	a.app = exec.CommandContext(appCtx, binaryPath, fmt.Sprintf("--root=%s", a.appRoot), fmt.Sprintf("--address=%s", a.appAddr))
	a.app.Stdout = os.Stdout
	a.app.Stderr = os.Stderr

	err := a.app.Start()
	if err != nil {
		return err
	}

	v, err := a.getAppVersion()
	if err != nil {
		return err
	}

	log.Info().Msgf("Started p2p app %q", v)
	return nil
}

func (a *LabAgent) getAppVersion() (string, error) {
	binaryPath := filepath.Join(a.root, LabAppBinary)

	buf := new(bytes.Buffer)
	cmd := exec.Command(binaryPath, "--version")
	cmd.Stdout = buf

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
