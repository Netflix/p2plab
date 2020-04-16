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

package buildrouter

import (
	"context"
	"io"
	"net/http"
	"net/url"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/daemon"
	"github.com/Netflix/p2plab/downloaders"
	"github.com/Netflix/p2plab/metadata"
	"github.com/google/uuid"
)

type router struct {
	db       metadata.DB
	uploader p2plab.Uploader
	fs       *downloaders.Downloaders
}

func New(db metadata.DB, uploader p2plab.Uploader, fs *downloaders.Downloaders) daemon.Router {
	return &router{db, uploader, fs}
}

func (b *router) Routes() []daemon.Route {
	return []daemon.Route{
		// GET
		daemon.NewGetRoute("/builds/json", b.getBuilds),
		daemon.NewGetRoute("/builds/{id}/json", b.getBuildByID),
		daemon.NewGetRoute("/builds/{id}/download", b.getBuildDownload),
		// POST
		daemon.NewPostRoute("/builds/upload", b.postBuildsUpload),
	}
}

func (b *router) getBuilds(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	builds, err := b.db.ListBuilds(ctx)
	if err != nil {
		return err
	}

	return daemon.WriteJSON(w, &builds)
}

func (b *router) getBuildByID(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	id := vars["id"]
	build, err := b.db.GetBuild(ctx, id)
	if err != nil {
		return err
	}

	return daemon.WriteJSON(w, &build)
}

func (b *router) getBuildDownload(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	id := vars["id"]
	build, err := b.db.GetBuild(ctx, id)
	if err != nil {
		return err
	}

	u, err := url.Parse(build.Link)
	if err != nil {
		return err
	}

	downloader, err := b.fs.Get(u.Scheme)
	if err != nil {
		return err
	}

	rc, err := downloader.Download(ctx, build.Link)
	if err != nil {
		return err
	}
	defer rc.Close()

	_, err = io.Copy(w, rc)
	return err
}

func (b *router) postBuildsUpload(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	link, err := b.uploader.Upload(ctx, r.Body)
	if err != nil {
		return err
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return err
	}

	build, err := b.db.CreateBuild(ctx, metadata.Build{
		ID:   id.String(),
		Link: link,
	})
	if err != nil {
		return err
	}

	return daemon.WriteJSON(w, &build)
}
