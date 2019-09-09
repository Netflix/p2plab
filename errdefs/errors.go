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

package errdefs

import (
	"context"

	"github.com/pkg/errors"
)

var (
	// ErrAlreadyExists is returned when a resource already exists.
	ErrAlreadyExists = errors.New("already exists")

	// ErrNotFound is returned when a resource is not found.
	ErrNotFound = errors.New("not found")

	// ErrInvalidArgument is returned when a invalid argument was given.
	ErrInvalidArgument = errors.New("invalid argument")

	ErrUnavailable = errors.New("unavailable")
)

func IsAlreadyExists(err error) bool {
	return errors.Cause(err) == ErrAlreadyExists
}

func IsNotFound(err error) bool {
	return errors.Cause(err) == ErrNotFound
}

func IsInvalidArgument(err error) bool {
	return errors.Cause(err) == ErrInvalidArgument
}

func IsUnavailable(err error) bool {
	return errors.Cause(err) == ErrUnavailable
}

func IsCancelled(err error) bool {
	return errors.Cause(err) == context.Canceled
}
