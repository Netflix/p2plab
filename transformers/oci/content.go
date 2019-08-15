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
	"context"
	"io"
	"io/ioutil"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/pkg/digestconv"
	"github.com/containerd/containerd/content"
	files "github.com/ipfs/go-ipfs-files"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

type provider struct {
	peer p2plab.Peer
}

func NewProvider(peer p2plab.Peer) content.Provider {
	return &provider{peer}
}

// ReaderAt only requires desc.Digest to be set.
// Other fields in the descriptor may be used internally for resolving
// the location of the actual data.
func (p *provider) ReaderAt(ctx context.Context, desc ocispec.Descriptor) (content.ReaderAt, error) {
	c, err := digestconv.DigestToCid(desc.Digest)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert digest %q to cid", desc.Digest)
	}

	nd, err := p.peer.Get(ctx, c)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get file %q", c)
	}

	r := files.ToFile(nd)
	if r == nil {
		return nil, errors.New("expected node to be a unixfs file")
	}

	return &sizeReaderAt{
		size: desc.Size,
		rc:   r,
	}, nil
}

type sizeReaderAt struct {
	size int64
	rc   io.ReadCloser
	n    int64
}

func (ra *sizeReaderAt) ReadAt(p []byte, offset int64) (n int, err error) {
	if offset < ra.n {
		return 0, errors.New("invalid offset")
	}
	diff := offset - ra.n
	written, err := io.CopyN(ioutil.Discard, ra.rc, diff)
	ra.n += written
	if err != nil {
		return int(written), err
	}

	n, err = ra.rc.Read(p)
	ra.n += int64(n)
	return
}

func (ra *sizeReaderAt) Size() int64 {
	return ra.size
}

func (ra *sizeReaderAt) Close() error {
	return ra.rc.Close()
}
