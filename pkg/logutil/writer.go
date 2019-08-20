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
	"bufio"
	"io"

	"github.com/rs/zerolog"
)

type logWriter struct {
	pipew   io.WriteCloser
	scanner *bufio.Scanner
}

func NewWriter(logger *zerolog.Logger, level zerolog.Level) io.WriteCloser {
	piper, pipew := io.Pipe()
	scanner := bufio.NewScanner(piper)
	go func() {
		defer piper.Close()
		for scanner.Scan() {
			logger.WithLevel(level).Msg(scanner.Text())
		}
	}()

	return &logWriter{pipew, scanner}
}

func (a *logWriter) Write(p []byte) (int, error) {
	return a.pipew.Write(p)
}

func (a *logWriter) Close() error {
	a.pipew.Close()
	return a.scanner.Err()
}
