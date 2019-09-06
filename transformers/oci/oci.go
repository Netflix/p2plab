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

package oci

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/errdefs"
	"github.com/Netflix/p2plab/pkg/digestconv"
	"github.com/Netflix/p2plab/pkg/traceutil"
	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/content/local"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/platforms"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	cid "github.com/ipfs/go-cid"
	"github.com/ipfs/go-ipfs/dagutils"
	ipld "github.com/ipfs/go-ipld-format"
	unixfs "github.com/ipfs/go-unixfs"
	multihash "github.com/multiformats/go-multihash"
	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	bolt "go.etcd.io/bbolt"
)

type transformer struct {
	root     string
	db       *bolt.DB
	store    content.Store
	resolver remotes.Resolver
}

func New(root string, client *http.Client) (p2plab.Transformer, error) {
	store, err := local.NewStore(filepath.Join(root, "store"))
	if err != nil {
		return nil, err
	}

	path := filepath.Join(root, "meta.db")
	db, err := bolt.Open(path, 0644, nil)
	if err != nil {
		return nil, err
	}

	resolver := docker.NewResolver(docker.ResolverOptions{
		Client: client,
	})

	return &transformer{
		root:     root,
		db:       db,
		store:    store,
		resolver: resolver,
	}, nil
}

func (t *transformer) Close() error {
	return t.db.Close()
}

func (t *transformer) Transform(ctx context.Context, p p2plab.Peer, source string, opts ...p2plab.AddOption) (cid.Cid, error) {
	span, ctx := traceutil.StartSpanFromContext(ctx, "transformer.Transform")
	defer span.Finish()
	span.SetTag("peer", p.Host().ID().String())
	span.SetTag("source", source)

	zerolog.Ctx(ctx).Info().Str("source", source).Msg("Resolving OCI reference")
	name, desc, err := t.resolver.Resolve(ctx, source)
	if err != nil {
		return cid.Undef, errors.Wrapf(err, "failed to resolve %q", source)
	}
	zerolog.Ctx(ctx).Info().Str("source", source).Str("digest", desc.Digest.String()).Msg("Resolved reference to digest")

	target, err := t.get(desc.Digest)
	if err != nil && !errdefs.IsNotFound(err) {
		return cid.Undef, errors.Wrapf(err, "failed to look for cached transform")
	}

	if errdefs.IsNotFound(err) {
		fetcher, err := t.resolver.Fetcher(ctx, name)
		if err != nil {
			return cid.Undef, errors.Wrapf(err, "failed to create fetcher for %q", name)
		}

		zerolog.Ctx(ctx).Info().Str("digest", desc.Digest.String()).Msg("Converting manifest recursively to IPLD DAG")
		target, err = Convert(ctx, p, fetcher, t.store, desc, opts...)
		if err != nil {
			return cid.Undef, errors.Wrapf(err, "failed to convert %q", name)
		}

		err = t.put(desc.Digest, target)
		if err != nil {
			return cid.Undef, errors.Wrapf(err, "failed to put cached transform")
		}
	}

	zerolog.Ctx(ctx).Info().Str("target", target.Digest.String()).Msg("Constructing Unixfs directory over manifest blobs")
	nd, err := ConstructDAGFromManifest(ctx, p, target, opts...)
	if err != nil {
		return cid.Undef, err
	}

	return nd.Cid(), nil
}

func Convert(ctx context.Context, peer p2plab.Peer, fetcher remotes.Fetcher, store content.Store, desc ocispec.Descriptor, opts ...p2plab.AddOption) (target ocispec.Descriptor, err error) {
	// Get all the children for a descriptor from a provider.
	childrenHandler := images.ChildrenHandler(store)
	// Filter manifests by platform.
	childrenHandler = images.FilterPlatforms(childrenHandler, platforms.Only(specs.Platform{
		OS:           "linux",
		Architecture: "amd64",
	}))
	// Convert each child into a IPLD merkle tree.
	childrenHandler = DispatchConvertHandler(childrenHandler, peer, fetcher, store, opts...)
	// Build manifest from converted children.
	childrenHandler = BuildManifestHandler(childrenHandler, peer, store, func(desc ocispec.Descriptor) {
		target = desc
	}, opts...)

	handler := images.Handlers(
		remotes.FetchHandler(store, fetcher),
		childrenHandler,
	)

	err = images.Dispatch(ctx, handler, nil, desc)
	if err != nil {
		return ocispec.Descriptor{}, errors.Wrap(err, "failed to dispatch")
	}

	return target, nil
}

func DispatchConvertHandler(f images.HandlerFunc, peer p2plab.Peer, fetcher remotes.Fetcher, store content.Store, opts ...p2plab.AddOption) images.HandlerFunc {
	return func(ctx context.Context, desc ocispec.Descriptor) (subdescs []ocispec.Descriptor, err error) {
		children, err := f(ctx, desc)
		if err != nil {
			return children, err
		}

		conversions := make(map[digest.Digest]ocispec.Descriptor)
		handler := ConvertHandler(conversions, peer, fetcher, store, opts...)
		err = images.Dispatch(ctx, handler, nil, children...)
		if err != nil {
			return children, errors.Wrap(err, "failed to sub-dispatch")
		}

		for i, desc := range children {
			children[i] = conversions[desc.Digest]
		}

		return children, nil
	}
}

func ConvertHandler(conversions map[digest.Digest]ocispec.Descriptor, peer p2plab.Peer, fetcher remotes.Fetcher, store content.Store, opts ...p2plab.AddOption) images.HandlerFunc {
	return func(ctx context.Context, desc ocispec.Descriptor) (subdescs []ocispec.Descriptor, err error) {
		var (
			target ocispec.Descriptor
		)
		switch desc.MediaType {
		case images.MediaTypeDockerSchema2Manifest, ocispec.MediaTypeImageManifest,
			images.MediaTypeDockerSchema2ManifestList, ocispec.MediaTypeImageIndex:

			target, err = Convert(ctx, peer, fetcher, store, desc)

		case images.MediaTypeDockerSchema2Layer, images.MediaTypeDockerSchema2LayerGzip,
			images.MediaTypeDockerSchema2LayerForeign, images.MediaTypeDockerSchema2LayerForeignGzip,
			images.MediaTypeDockerSchema2Config, ocispec.MediaTypeImageConfig,
			ocispec.MediaTypeImageLayer, ocispec.MediaTypeImageLayerGzip,
			ocispec.MediaTypeImageLayerNonDistributable, ocispec.MediaTypeImageLayerNonDistributableGzip,
			images.MediaTypeContainerd1Checkpoint, images.MediaTypeContainerd1CheckpointConfig:

			target = desc

			rc, err := fetcher.Fetch(ctx, desc)
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			target.Digest, err = AddBlob(ctx, peer, rc, opts...)
		}
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert %q [%s]", desc.Digest, desc.MediaType)
		}

		if zerolog.Ctx(ctx).GetLevel() == zerolog.DebugLevel {
			c, err := digestconv.DigestToCid(target.Digest)
			if err != nil {
				return nil, err
			}
			zerolog.Ctx(ctx).Debug().Str("mediaType", desc.MediaType).Str("source", desc.Digest.String()).Str("cid", c.String()).Int64("size", desc.Size).Msg("Added blob to peer")
		}

		conversions[desc.Digest] = target
		return nil, nil
	}
}

func BuildManifestHandler(f images.HandlerFunc, peer p2plab.Peer, provider content.Provider, callback func(ocispec.Descriptor), opts ...p2plab.AddOption) images.HandlerFunc {
	return func(ctx context.Context, desc ocispec.Descriptor) (subdescs []ocispec.Descriptor, err error) {
		children, err := f(ctx, desc)
		if err != nil {
			return children, err
		}

		var data interface{}
		switch desc.MediaType {
		case images.MediaTypeDockerSchema2Manifest, ocispec.MediaTypeImageManifest:
			p, err := content.ReadBlob(ctx, provider, desc)
			if err != nil {
				return nil, err
			}

			var manifest ocispec.Manifest
			err = json.Unmarshal(p, &manifest)
			if err != nil {
				return nil, err
			}

			manifest.Config = children[0]
			manifest.Layers = children[1:]
			data = &manifest

		case images.MediaTypeDockerSchema2ManifestList, ocispec.MediaTypeImageIndex:
			p, err := content.ReadBlob(ctx, provider, desc)
			if err != nil {
				return nil, err
			}

			var index ocispec.Index
			err = json.Unmarshal(p, &index)
			if err != nil {
				return nil, err
			}

			index.Manifests = children
			data = &index
		}

		blob, err := json.MarshalIndent(data, "", "    ")
		if err != nil {
			return nil, err
		}

		desc.Size = int64(len(blob))
		desc.Digest, err = AddBlob(ctx, peer, bytes.NewReader(blob), opts...)
		if err != nil {
			return nil, errors.Wrap(err, "failed to write blob")
		}

		callback(desc)
		return nil, nil
	}
}

func AddBlob(ctx context.Context, peer p2plab.Peer, r io.Reader, opts ...p2plab.AddOption) (digest.Digest, error) {
	n, err := peer.Add(ctx, r, opts...)
	if err != nil {
		return "", err
	}

	c := n.Cid()
	dgst, err := digestconv.CidToDigest(c)
	if err != nil {
		return "", err
	}

	return dgst, nil
}

func ConstructDAGFromManifest(ctx context.Context, p p2plab.Peer, image ocispec.Descriptor, opts ...p2plab.AddOption) (ipld.Node, error) {
	settings := p2plab.AddSettings{
		HashFunc: "sha2-256",
	}
	for _, opt := range opts {
		err := opt(&settings)
		if err != nil {
			return nil, err
		}
	}

	provider := NewProvider(p)
	manifest, err := images.Manifest(ctx, provider, image, platforms.Only(specs.Platform{
		OS:           "linux",
		Architecture: "amd64",
	}))
	if err != nil {
		return nil, err
	}

	root := unixfs.EmptyDirNode()
	root.SetCidBuilder(cid.V1Builder{MhType: multihash.Names[settings.HashFunc]})

	dserv := p.DAGService()
	e := dagutils.NewDagEditor(root, dserv)

	descs := []ocispec.Descriptor{manifest.Config}
	descs = append(descs, manifest.Layers...)

	for _, desc := range descs {
		c, err := digestconv.DigestToCid(desc.Digest)
		if err != nil {
			return nil, err
		}

		nd, err := dserv.Get(ctx, c)
		if err != nil {
			return nil, err
		}

		err = root.AddNodeLink(desc.Digest.String(), nd)
		if err != nil {
			return nil, err
		}
	}

	err = dserv.Add(ctx, root)
	if err != nil {
		return nil, err
	}

	return e.Finalize(ctx, dserv)
}
