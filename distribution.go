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

package p2plab

import (
	"context"
	"io"

	"github.com/Netflix/p2plab/metadata"
)

// Builder compiles the peer to hotswap the underlying implementation on a live
// cluster.
type Builder interface {
	// Init initializes the builder.
	Init(ctx context.Context) error

	// Resolve resolves a git-ref to a commit it can uniquely build.
	Resolve(ctx context.Context, ref string) (commit string, err error)

	// Build compiles the peer at the given commit.
	Build(ctx context.Context, commit string) (link string, err error)
}

// BuildAPI defines the API for build operations.
type BuildAPI interface {
	// Get returns a build.
	Get(ctx context.Context, id string) (Build, error)

	// List returns available builds.
	List(ctx context.Context) ([]Build, error)

	// Upload uploads a binary for a build.
	Upload(ctx context.Context, r io.Reader) (Build, error)
}

// Build is an compiled peer ready to be deployed.
type Build interface {
	// ID returns a uniquely identifiable string.
	ID() string

	Metadata() metadata.Build

	// Open creates a reader for the build's binary.
	Open(ctx context.Context) (io.ReadCloser, error)
}

// Uploader uploads artifacts to an external distribution mechanism.
type Uploader interface {
	// Upload uploads artifacts to a registry.
	Upload(ctx context.Context, r io.Reader) (link string, err error)

	Close() error
}

// Download downlaods artifacts from an external distribution mechanism.
type Downloader interface {
	// Download downloads an artifact with an abstract link.
	Download(ctx context.Context, link string) (io.ReadCloser, error)
}
