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
	"testing"

	cid "github.com/ipfs/go-cid"
	util "github.com/ipfs/go-ipfs-util"
	digest "github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

func TestDigestToCid(t *testing.T) {
	data := []byte("foobar")
	expected := cid.NewCidV1(cid.DagProtobuf, util.Hash(data))
	actual, err := DigestToCid(digest.FromBytes(data))
	require.NoError(t, err)
	require.Equal(t, expected.String(), actual.String())
}

func TestCidToDigest(t *testing.T) {
	data := []byte("foobar")
	expected := digest.FromBytes(data)
	actual, err := CidToDigest(cid.NewCidV1(cid.DagProtobuf, util.Hash(data)))
	require.NoError(t, err)
	require.Equal(t, expected.String(), actual.String())
}
