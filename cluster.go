package p2plab

import "context"

type ClusterAPI interface {
	Create(ctx context.Context, opts ...CreateClusterOption) (Cluster, error)

	Get(ctx context.Context, id string) (Cluster, error)

	List(ctx context.Context) ([]Cluster, error)
}

type Cluster interface {
	Remove(ctx context.Context) error

	Query(ctx context.Context, q Query) (NodeSet, error)

	Update(ctx context.Context, commit string) error
}

type CreateClusterOption func(CreateClusterSettings) error

type CreateClusterSettings struct {
}
