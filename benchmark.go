package p2plab

import (
	"context"
	"io"
)

type BenchmarkAPI interface {
	Get(ctx context.Context, id string) (Benchmark, error)

	List(ctx context.Context) ([]Benchmark, error)
}

type Benchmark interface {
	Start(ctx context.Context) error

	Cancel(ctx context.Context) error

	Report(ctx context.Context) (Report, error)

	Logs(ctx context.Context, opt ...LogsOption) (io.ReadCloser, error)
}

type Report interface {
}

type LogsOption func(LogsSettings) error

type LogsSettings struct {
	Tail bool
}
