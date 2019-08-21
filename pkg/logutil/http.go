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

package logutil

import (
	"context"
	"io"
	"net/http"
	"os"

	"github.com/rs/zerolog"
)

func WithResponseLogger(ctx context.Context, w http.ResponseWriter) (context.Context, *zerolog.Logger) {
	multiwriter := io.MultiWriter(os.Stderr, NewWriteFlusher(w))
	logger := zerolog.Ctx(ctx).Output(multiwriter)
	ctx = logger.WithContext(WithLogWriter(ctx, multiwriter))
	return ctx, &logger
}

type WriteFlusher struct {
	w io.Writer
	f http.Flusher
}

func NewWriteFlusher(w io.Writer) *WriteFlusher {
	wf := WriteFlusher{w: w}
	f, ok := w.(http.Flusher)
	if ok {
		wf.f = f
	}
	return &wf
}

func (wf *WriteFlusher) Write(p []byte) (int, error) {
	n, err := wf.w.Write(p)
	if err != nil {
		return n, err
	}
	wf.f.Flush()
	return n, err
}
