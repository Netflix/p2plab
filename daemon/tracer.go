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

package daemon

import (
	"io"
	"log"
	"os"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	jaeger "github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
)

func NewTracer(service string, logger jaeger.Logger) (opentracing.Tracer, io.Closer) {
	tracerAddr := os.Getenv("JAEGER_TRACE")
	if tracerAddr != "" {
		cfg := config.Configuration{
			Sampler: &config.SamplerConfig{
				Type:  "const",
				Param: 1,
			},
			Reporter: &config.ReporterConfig{
				LogSpans:            true,
				BufferFlushInterval: time.Second,
				LocalAgentHostPort:  tracerAddr,
			},
		}
		tracer, closer, err := cfg.New(
			service,
			config.Logger(logger),
		)
		if err != nil {
			log.Fatal(err)
		}

		opentracing.SetGlobalTracer(tracer)
		return tracer, closer
	}

	return opentracing.NoopTracer{}, &nopCloser{}
}

type nopCloser struct{}

func (*nopCloser) Close() error {
	return nil
}
