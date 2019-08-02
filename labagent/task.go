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

package labagent

import (
	"context"
	"time"
)

type TaskRequest struct {
	Type TaskType
	Args []string
}

type TaskType string

var (
	TaskGet TaskType = "get"
)

type TaskResponse struct {
	Err         error
	TimeElapsed time.Duration
}

func (a *LabAgent) sendTask(ctx context.Context, req TaskRequest) (TaskResponse, error) {
	var resp TaskResponse
	err := a.appEncoder.Encode(&req)
	if err != nil {
		return resp, err
	}

	err = a.appDecoder.Decode(&resp)
	if err != nil {
		return resp, err
	}

	return resp, nil
}
