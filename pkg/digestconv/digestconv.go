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

package digestconv

import (
	"encoding/hex"

	cid "github.com/ipfs/go-cid"
	multihash "github.com/multiformats/go-multihash"
	digest "github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

func DigestToCid(dgst digest.Digest) (cid.Cid, error) {
	data, err := hex.DecodeString(dgst.Hex())
	if err != nil {
		return cid.Cid{}, errors.Wrap(err, "failed to decode digest hex")
	}

	encoded, err := multihash.Encode(data[:32], multihash.SHA2_256)
	if err != nil {
		return cid.Cid{}, errors.Wrap(err, "failed to encode digest as SHA256 multihash")
	}

	return cid.NewCidV1(cid.DagProtobuf, multihash.Multihash(encoded)), nil
}

func CidToDigest(c cid.Cid) (digest.Digest, error) {
	decoded, err := multihash.Decode(c.Hash())
	if err != nil {
		return "", err
	}

	return digest.NewDigestFromBytes(digest.Canonical, decoded.Digest), nil
}
