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

	cid "github.com/ipfs/go-cid"
)

// Transformer defines a way to convert an external resource into IPFS DAGs.
type Transformer interface {
	// Transform adds a resource defined by source into an IPFS DAG stored in
	// peer.
	Transform(ctx context.Context, peer Peer, source string, opts ...AddOption) (cid.Cid, error)

	// Close releases any resources held by the Transformer.
	Close() error
}
