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

package metadata

import (
	"regexp"

	"github.com/Netflix/p2plab/errdefs"
	"github.com/pkg/errors"
)

var (
	ClusterIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,32}`)
)

func ValidateClusterID(id string) error {
	match := ClusterIDPattern.MatchString(id)
	if !match {
		return errors.Wrapf(errdefs.ErrInvalidArgument, "cluster id must match %q", ClusterIDPattern)
	}
	return nil
}
