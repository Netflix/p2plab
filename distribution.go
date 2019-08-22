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
	"io"
)

type Builder interface {
	Build(ctx context.Context, ref string) (io.Reader, error)
}

type Uploader interface {
	Upload(ctx context.Context, r io.Reader) (ref string, err error)
}

type Downloader interface {
	Download(ctx context.Context, ref string) (io.ReadCloser, error)
}
