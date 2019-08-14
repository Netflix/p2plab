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

package terraform

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
)

type EC2Instance struct {
	InstanceId   string `json:"InstanceId"`
	InstanceType string `json:"InstanceType"`
	PrivateIp    string `json:"PrivateIpAddress"`
}

func DiscoverInstances(ctx context.Context, asg, region string) ([]EC2Instance, error) {
	asgStdout := new(bytes.Buffer)
	err := awscliWithStdio(ctx, asgStdout, nil, "autoscaling", "describe-auto-scaling-groups",
		"--query", "AutoScalingGroups[].Instances[].InstanceId",
		"--output", "json",
		"--region", region,
		"--auto-scaling-group-names", asg,
	)
	if err != nil {
		return nil, err
	}

	var instanceIds []string
	err = json.NewDecoder(asgStdout).Decode(&instanceIds)
	if err != nil {
		return nil, err
	}

	instancesStdout := new(bytes.Buffer)
	err = awscliWithStdio(ctx, instancesStdout, nil, append([]string{"ec2", "describe-instances",
		"--query", "Reservations[].Instances[]",
		"--output", "json",
		"--region", region,
		"--instance-ids"}, instanceIds...)...,
	)
	if err != nil {
		return nil, err
	}

	var instances []EC2Instance
	err = json.NewDecoder(instancesStdout).Decode(&instances)
	if err != nil {
		return nil, err
	}

	return instances, nil
}

func awscli(ctx context.Context, args ...string) error {
	return awscliWithStdio(ctx, os.Stdout, os.Stderr, args...)
}

func awscliWithStdio(ctx context.Context, stdout, stderr io.Writer, args ...string) error {
	cmd := exec.CommandContext(ctx, "aws", args...)
	cmd.Stdin = nil
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
