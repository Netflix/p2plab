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

package builder

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/errdefs"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/pkg/logutil"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type builder struct {
	root         string
	bareRepoPath string
	db           metadata.DB
	uploader     p2plab.Uploader
}

func New(root string, db metadata.DB, uploader p2plab.Uploader) (p2plab.Builder, error) {
	err := os.MkdirAll(root, 0711)
	if err != nil {
		return nil, err
	}

	return &builder{
		root:     root,
		db:       db,
		uploader: uploader,
	}, nil
}

func (b *builder) Init(ctx context.Context) error {
	bareRepoPath := filepath.Join(b.root, "p2plab.git")
	_, err := os.Stat(bareRepoPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		// Clone a bare repository to allow arbitrary fetching of commits.
		err = b.git(ctx, b.root, "clone", "--bare", "https://github.com/Netflix/p2plab.git")
		if err != nil {
			return err
		}

		// Starting with git 2.5.0 onward, we need to enable the server side bare repo
		// to allow fetching of specific SHA1s.
		err = b.git(ctx, bareRepoPath, "config", "--bool", "--add", "uploadpack.allowReachableSHA1InWant", "true")
		if err != nil {
			return err
		}

		// Allows `git fetch origin` to fetch all upstream changes to all branches.
		err = b.git(ctx, bareRepoPath, "config", "--add", "remote.origin.fetch", "refs/heads/*:refs/heads/*")
		if err != nil {
			return err
		}
	}

	b.bareRepoPath, err = filepath.Abs(bareRepoPath)
	if err != nil {
		return err
	}

	return nil
}

func (b *builder) Resolve(ctx context.Context, ref string) (commit string, err error) {
	// Resolve a remote repository's branches and tags.
	//
	// $ git ls-remote https://github.com/Netflix/p2plab.git HEAD
	// d29b2cd10302df3197924297441d1ade74b3fc44	HEAD
	buf := new(bytes.Buffer)
	err = b.execWithStdio(ctx, b.root, buf, ioutil.Discard, "git", "ls-remote", "https://github.com/Netflix/p2plab.git", ref)
	if err != nil {
		return "", err
	}
	output := strings.TrimSpace(buf.String())

	// If nothing is resolved, assume the ref is a commit ref.
	if len(output) == 0 {
		return ref, nil
	}

	parts := strings.Split(output, "\t")
	resolved := parts[0]
	zerolog.Ctx(ctx).Debug().Str("ref", ref).Str("resolved", resolved).Msg("Resolved remote git ref")
	return resolved, nil
}

func (b *builder) Build(ctx context.Context, commit string) (link string, err error) {
	build, err := b.db.GetBuild(ctx, commit)
	if err == nil {
		return build.Link, nil
	}

	if !errdefs.IsNotFound(err) {
		return "", errors.Wrap(err, "failed to get build from db")
	}

	f, dir, err := b.buildCommit(ctx, commit)
	if err != nil {
		rmErr := os.RemoveAll(dir)
		if rmErr != nil {
			zerolog.Ctx(ctx).Debug().Str("dir", dir).Msg("failed to cleanup build dir")
		}
		return "", errors.Wrapf(err, "failed to build commit %q", commit)
	}
	defer os.RemoveAll(dir)
	defer f.Close()

	link, err = b.uploader.Upload(ctx, f)
	if err != nil {
		return "", errors.Wrap(err, "failed to upload build")
	}

	// Create build, ignoring if it already exists because of a parallel build
	// request.
	build, err = b.db.CreateBuild(ctx, metadata.Build{
		ID:   commit,
		Link: link,
	})
	if err != nil && !errdefs.IsAlreadyExists(err) {
		return "", err
	}

	return link, nil
}

func (b *builder) buildCommit(ctx context.Context, commit string) (f *os.File, dir string, err error) {
	dir, err = ioutil.TempDir(b.root, commit)
	if err != nil {
		return nil, dir, err
	}

	err = b.git(ctx, b.bareRepoPath, "fetch", "--force", "origin")
	if err != nil {
		return nil, dir, err
	}

	shallowRepoPath := filepath.Join(dir, "p2plab")
	err = os.MkdirAll(shallowRepoPath, 0711)
	if err != nil {
		return nil, dir, err
	}

	err = b.git(ctx, shallowRepoPath, "init")
	if err != nil {
		return nil, dir, err
	}

	// Set origin of shallow repo to be the bare repo on disk.
	err = b.git(ctx, shallowRepoPath, "remote", "add", "origin", b.bareRepoPath)
	if err != nil {
		return nil, dir, err
	}

	// Retrieve commit metadata from bare repo.
	err = b.git(ctx, shallowRepoPath, "fetch", "--depth", "1", "origin", commit)
	if err != nil {
		return nil, dir, err
	}

	// Check out commit into shallow repo working directory.
	err = b.git(ctx, shallowRepoPath, "reset", "--hard", commit)
	if err != nil {
		return nil, dir, err
	}

	err = b.goBuild(ctx, shallowRepoPath, "-o", "build", "./cmd/labapp")
	if err != nil {
		return nil, dir, err
	}

	f, err = os.Open(filepath.Join(shallowRepoPath, "build"))
	if err != nil {
		return nil, dir, err
	}

	return f, dir, nil
}

func (b *builder) goBuild(ctx context.Context, workDir string, args ...string) error {
	logger := zerolog.Ctx(ctx).With().Strs("exec", args).Logger()
	logWriter := logutil.NewWriter(&logger, zerolog.DebugLevel)
	defer logWriter.Close()

	execArgs := append([]string{"go", "build"}, args...)
	return b.execWithStdio(ctx, workDir, logWriter, logWriter, execArgs...)
}

func (b *builder) git(ctx context.Context, workDir string, args ...string) error {
	logger := zerolog.Ctx(ctx).With().Strs("exec", args).Logger()
	logWriter := logutil.NewWriter(&logger, zerolog.DebugLevel)
	defer logWriter.Close()

	execArgs := append([]string{"git"}, args...)
	return b.execWithStdio(ctx, workDir, logWriter, logWriter, execArgs...)
}

func (b *builder) execWithStdio(ctx context.Context, workDir string, stdout, stderr io.Writer, args ...string) error {
	if len(args) == 0 {
		panic("exec called with zero args")
	}

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = workDir
	cmd.Stdin = nil
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
