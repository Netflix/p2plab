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
	"encoding/binary"
	"fmt"

	"github.com/Netflix/p2plab/errdefs"
	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	bolt "go.etcd.io/bbolt"
)

var (
	bucketKeyDigest    = []byte("digest")
	bucketKeyMediaType = []byte("mediaType")
	bucketKeySize      = []byte("size")
)

func (t *transformer) get(dgst digest.Digest) (desc ocispec.Descriptor, err error) {
	err = t.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(dgst.String()))
		if bkt == nil {
			return errdefs.ErrNotFound
		}

		return bkt.ForEach(func(k, v []byte) error {
			if v == nil {
				return nil
			}

			switch string(k) {
			case string(bucketKeyDigest):
				desc.Digest = digest.Digest(v)
			case string(bucketKeyMediaType):
				desc.MediaType = string(v)
			case string(bucketKeySize):
				desc.Size, _ = binary.Varint(v)
			}

			return nil
		})
	})
	if err != nil {
		return desc, err
	}

	return desc, nil
}

func (t *transformer) put(dgst digest.Digest, desc ocispec.Descriptor) error {
	return t.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(dgst.String()))
		if bkt != nil {
			err := tx.DeleteBucket([]byte(dgst.String()))
			if err != nil {
				return err
			}
		}

		var err error
		bkt, err = tx.CreateBucket([]byte(dgst.String()))
		if err != nil {
			return err
		}

		sizeEncoded, err := encodeInt(desc.Size)
		if err != nil {
			return err
		}

		for _, v := range [][2][]byte{
			{bucketKeyDigest, []byte(desc.Digest)},
			{bucketKeyMediaType, []byte(desc.MediaType)},
			{bucketKeySize, sizeEncoded},
		} {
			if err := bkt.Put(v[0], v[1]); err != nil {
				return err
			}
		}

		return nil
	})
}

func encodeInt(i int64) ([]byte, error) {
	var (
		buf     [binary.MaxVarintLen64]byte
		encoded = buf[:]
	)
	encoded = encoded[:binary.PutVarint(encoded, i)]

	if len(encoded) == 0 {
		return nil, fmt.Errorf("failed encoding integer = %v", i)
	}
	return encoded, nil
}
