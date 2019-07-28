package p2plab

import "context"

type BenchmarkAPI interface {
	Get(ctx context.Context, id string) (Benchmark, error)

	List(ctx context.Context) ([]Benchmark, error)
}

type Benchmark interface {
}
