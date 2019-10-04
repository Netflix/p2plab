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

package traceutil

import (
	"context"
	"io"
	"log"
	"os"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	jaeger "github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
)

type tracerKey struct{}

func WithTracer(ctx context.Context, tracer opentracing.Tracer) context.Context {
	return context.WithValue(ctx, tracerKey{}, tracer)
}

func Tracer(ctx context.Context) opentracing.Tracer {
	tracer, ok := ctx.Value(tracerKey{}).(opentracing.Tracer)
	if !ok {
		return opentracing.NoopTracer{}
	}
	return tracer
}

func StartSpanFromContext(ctx context.Context, operationName string, opts ...opentracing.StartSpanOption) (opentracing.Span, context.Context) {
	return opentracing.StartSpanFromContextWithTracer(ctx, Tracer(ctx), operationName, opts...)
}

func New(ctx context.Context, service string, logger jaeger.Logger) (context.Context, opentracing.Tracer, io.Closer) {
	tracerAddr := os.Getenv("JAEGER_TRACE")
	if tracerAddr != "" {
		cfg := config.Configuration{
			Sampler: &config.SamplerConfig{
				Type:  "const",
				Param: 1,
			},
			Reporter: &config.ReporterConfig{
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

		ctx = WithTracer(ctx, tracer)
		return ctx, tracer, closer
	}

	return ctx, opentracing.NoopTracer{}, &nopCloser{}
}

type nopCloser struct{}

func (*nopCloser) Close() error {
	return nil
}
