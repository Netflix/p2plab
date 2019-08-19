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

package actions

import (
	"context"
	"fmt"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/metadata"
	cid "github.com/ipfs/go-cid"
)

func Parse(objects map[string]cid.Cid, a string) (p2plab.Action, error) {
	return &dummyAction{objects[a].String()}, nil
}

type dummyAction struct {
	subject string
}

func (a *dummyAction) String() string {
	return fmt.Sprintf("get %q", a.subject)
}

func (a *dummyAction) Tasks(ctx context.Context, ns []p2plab.Node) (map[string]metadata.Task, error) {
	taskMap := make(map[string]metadata.Task)
	for _, n := range ns {
		taskMap[n.Metadata().ID] = metadata.Task{
			Type:    metadata.TaskGet,
			Subject: a.subject,
		}
	}
	return taskMap, nil
}
