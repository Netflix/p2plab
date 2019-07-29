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

import "context"

type NodeAPI interface {
	Get(ctx context.Context, id string) (Node, error)
}

type Node interface {
	SSH(ctx context.Context, opts ...SSHOpt) error
}

type NodeSet interface {
	Add(node Node)

	Remove(node Node)

	Slice() []Node

	Label(ctx context.Context, addLabels, removeLabels []string) error
}

type SSHOpt func(SSHSettings) error

type SSHSettings struct {
}
