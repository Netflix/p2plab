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
	"github.com/rs/zerolog"
	jaeger "github.com/uber/jaeger-client-go"
)

type jaegerLogger struct {
	logger *zerolog.Logger
}

func NewJaegerLogger(logger *zerolog.Logger) jaeger.Logger {
	return &jaegerLogger{logger}
}

type Logger interface {
	// Error logs a message at error priority
	Error(msg string)

	// Infof logs a message at info priority
	Infof(msg string, args ...interface{})
}

func (jl *jaegerLogger) Error(msg string) {
	jl.logger.Error().Msg(msg)
}

func (jl *jaegerLogger) Infof(msg string, args ...interface{}) {
	jl.logger.Info().Msgf(msg, args...)
}
