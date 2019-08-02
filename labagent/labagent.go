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
	"syscall"
	"time"

	"github.com/Netflix/p2plab/labd"
	"github.com/Netflix/p2plab/pkg/httputil"
	"github.com/containerd/fifo"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

var (
	LabAppBinary = "labapp"
	LabAppFifo   = "labapp-fifo"
)

type LabAgent struct {
	root       string
	addr       string
	router     *mux.Router
	httpClient *http.Client
	app        *exec.Cmd
	appCancel  func()
	appFifo    io.ReadWriteCloser
	appEncoder *json.Encoder
	appDecoder *json.Decoder
}

func New(root, addr string) (*LabAgent, error) {
	r := mux.NewRouter().UseEncodedPath().StrictSlash(true)
	la := &LabAgent{
		root:   root,
		addr:   addr,
		router: r,
		httpClient: &http.Client{
			Transport: &http.Transport{
				Proxy:             http.ProxyFromEnvironment,
				DisableKeepAlives: true,
			},
		},
	}
	la.registerRoutes(r)

	err := la.update(context.TODO(), "")
	if err != nil {
		return nil, err
	}

	return la, nil
}

func (a *LabAgent) Serve(ctx context.Context) error {
	log.Info().Msgf("APIserver listening on %s", a.addr)
	s := &http.Server{
		Handler:      a.router,
		Addr:         a.addr,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	return s.ListenAndServe()
}

func (a *LabAgent) registerRoutes(r *mux.Router) {
	api := r.PathPrefix("/api/v0").Subrouter()

	api.Handle("/get", httputil.ErrorHandler{a.getHandler}).Methods("POST")
	api.Handle("/update", httputil.ErrorHandler{a.updateHandler}).Methods("POST")
}

func (a *LabAgent) getHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("labagent/get")

	req := TaskRequest{
		Type: TaskGet,
		Args: []string{r.FormValue("cid")},
	}

	resp, err := a.sendTask(r.Context(), req)
	if err != nil {
		return err
	}

	return labd.WriteJSON(w, &resp)
}

func (a *LabAgent) updateHandler(w http.ResponseWriter, r *http.Request) error {
	log.Info().Msg("labagent/update")
	return a.update(r.Context(), r.FormValue("url"))
}

func (a *LabAgent) update(ctx context.Context, url string) error {
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
	if a.appFifo != nil {
		err := a.appFifo.Close()
		if err != nil {
			return err
		}

		// Kill subprocess.
		a.appCancel()

		err = a.app.Wait()
		if err != nil {
			return err
		}
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

	fdPath := filepath.Join(a.root, LabAppFifo)

	var err error
	a.appFifo, err = fifo.OpenFifo(appCtx, fdPath, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_NONBLOCK, 0700)
	if err != nil {
		return err
	}
	a.appEncoder, a.appDecoder = json.NewEncoder(a.appFifo), json.NewDecoder(a.appFifo)

	binaryPath := filepath.Join(a.root, LabAppBinary)
	a.app = exec.CommandContext(appCtx, binaryPath, fmt.Sprintf("--fd-path=%s", fdPath))
	a.app.Stdout = os.Stdout
	a.app.Stderr = os.Stderr

	err = a.app.Start()
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
