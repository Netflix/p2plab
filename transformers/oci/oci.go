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

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/pkg/digestconv"
	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/platforms"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	cid "github.com/ipfs/go-cid"
	"github.com/ipfs/go-ipfs/dagutils"
	ipld "github.com/ipfs/go-ipld-format"
	unixfs "github.com/ipfs/go-unixfs"
	"github.com/moby/buildkit/util/contentutil"
	multihash "github.com/multiformats/go-multihash"
	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type transformer struct {
	resolver remotes.Resolver
}

func New() p2plab.Transformer {
	resolver := docker.NewResolver(docker.ResolverOptions{
		Client: http.DefaultClient,
	})

	return &transformer{
		resolver: resolver,
	}
}

func (t *transformer) Transform(ctx context.Context, p p2plab.Peer, source string, options []string) (cid.Cid, error) {
	name, desc, err := t.resolver.Resolve(ctx, source)
	if err != nil {
		return cid.Undef, errors.Wrapf(err, "failed to resolve %q", source)
	}

	fetcher, err := t.resolver.Fetcher(ctx, name)
	if err != nil {
		return cid.Undef, errors.Wrapf(err, "failed to create fetcher for %q", name)
	}

	buffer := contentutil.NewBuffer()
	target, err := Convert(ctx, p, fetcher, buffer, desc)
	if err != nil {
		return cid.Undef, errors.Wrapf(err, "failed to convert %q", name)
	}

	nd, err := ConstructDAGFromManifest(ctx, p, target)
	if err != nil {
		return cid.Undef, err
	}

	return nd.Cid(), nil
}

func Convert(ctx context.Context, peer p2plab.Peer, fetcher remotes.Fetcher, buffer contentutil.Buffer, desc ocispec.Descriptor) (target ocispec.Descriptor, err error) {
	// Get all the children for a descriptor from a provider.
	childrenHandler := images.ChildrenHandler(buffer)
	// Convert each child into a IPLD merkle tree.
	childrenHandler = DispatchConvertHandler(childrenHandler, peer, fetcher, buffer)
	// Build manifest from converted children.
	childrenHandler = BuildManifestHandler(childrenHandler, peer, buffer, func(desc ocispec.Descriptor) {
		target = desc
	})

	handler := images.Handlers(
		remotes.FetchHandler(buffer, fetcher),
		childrenHandler,
	)

	err = images.Dispatch(ctx, handler, nil, desc)
	if err != nil {
		return ocispec.Descriptor{}, errors.Wrap(err, "failed to dispatch")
	}

	return target, nil
}

func DispatchConvertHandler(f images.HandlerFunc, peer p2plab.Peer, fetcher remotes.Fetcher, buffer contentutil.Buffer) images.HandlerFunc {
	return func(ctx context.Context, desc ocispec.Descriptor) (subdescs []ocispec.Descriptor, err error) {
		children, err := f(ctx, desc)
		if err != nil {
			return children, err
		}

		conversions := make(map[digest.Digest]ocispec.Descriptor)
		handler := ConvertHandler(conversions, peer, fetcher, buffer)
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

func ConvertHandler(conversions map[digest.Digest]ocispec.Descriptor, peer p2plab.Peer, fetcher remotes.Fetcher, buffer contentutil.Buffer) images.HandlerFunc {
	return func(ctx context.Context, desc ocispec.Descriptor) (subdescs []ocispec.Descriptor, err error) {
		var (
			target ocispec.Descriptor
		)
		switch desc.MediaType {
		case images.MediaTypeDockerSchema2Manifest, ocispec.MediaTypeImageManifest,
			images.MediaTypeDockerSchema2ManifestList, ocispec.MediaTypeImageIndex:

			target, err = Convert(ctx, peer, fetcher, buffer, desc)

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

			target.Digest, err = AddBlob(ctx, peer, rc)
		}
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert %q [%s]", desc.Digest, desc.MediaType)
		}

		if log.Logger.GetLevel() == zerolog.DebugLevel {
			c, err := digestconv.DigestToCid(target.Digest)
			if err != nil {
				return nil, err
			}
			log.Debug().Str("mediaType", desc.MediaType).Str("source", desc.Digest.String()).Str("cid", c.String()).Int64("size", desc.Size).Msg("Added blob to peer")
		}

		conversions[desc.Digest] = target
		return nil, nil
	}
}

func BuildManifestHandler(f images.HandlerFunc, peer p2plab.Peer, provider content.Provider, callback func(ocispec.Descriptor)) images.HandlerFunc {
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
		desc.Digest, err = AddBlob(ctx, peer, bytes.NewReader(blob))
		if err != nil {
			return nil, errors.Wrap(err, "failed to write blob")
		}

		callback(desc)
		return nil, nil
	}
}

func AddBlob(ctx context.Context, peer p2plab.Peer, r io.Reader) (digest.Digest, error) {
	n, err := peer.Add(ctx, r)
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

func ConstructDAGFromManifest(ctx context.Context, p p2plab.Peer, image ocispec.Descriptor) (ipld.Node, error) {
	provider := NewProvider(p)
	manifest, err := images.Manifest(ctx, provider, image, platforms.Default())
	if err != nil {
		return nil, err
	}

	root := unixfs.EmptyDirNode()
	root.SetCidBuilder(cid.V1Builder{MhType: multihash.SHA2_256})

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
