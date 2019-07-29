package labd

import (
	"context"

	"github.com/Netflix/p2plab"
)

type client struct {
}

func NewClient() (p2plab.LabdAPI, error) {
	return &client{}, nil
}

func (c *client) Cluster() p2plab.ClusterAPI {
	return &clusterAPI{c}
}

func (c *client) Node() p2plab.NodeAPI {
	return &nodeAPI{c}
}

func (c *client) Scenario() p2plab.ScenarioAPI {
	return &scenarioAPI{c}
}

func (c *client) Benchmark() p2plab.BenchmarkAPI {
	return &benchmarkAPI{c}
}

type clusterAPI struct {
	c *client
}

func (capi *clusterAPI) Create(ctx context.Context, opts ...p2plab.CreateClusterOption) (p2plab.Cluster, error) {
	return nil, nil
}

func (capi *clusterAPI) Get(ctx context.Context, id string) (p2plab.Cluster, error) {
	return nil, nil
}

func (capi *clusterAPI) List(ctx context.Context) ([]p2plab.Cluster, error) {
	return nil, nil
}

type nodeAPI struct {
	c *client
}

func (napi *nodeAPI) Get(ctx context.Context, id string) (p2plab.Node, error) {
	return nil, nil
}

func (napi *nodeAPI) List(ctx context.Context) ([]p2plab.Node, error) {
	return nil, nil
}

type scenarioAPI struct {
	c *client
}

func (sapi *scenarioAPI) Create(ctx context.Context, name string, sdef p2plab.ScenarioDefinition) (p2plab.Scenario, error) {
	return nil, nil
}

func (sapi *scenarioAPI) Get(ctx context.Context, id string) (p2plab.Scenario, error) {
	return nil, nil
}

func (sapi *scenarioAPI) List(ctx context.Context) ([]p2plab.Scenario, error) {
	return nil, nil
}

type benchmarkAPI struct {
	c *client
}

func (sapi *benchmarkAPI) Get(ctx context.Context, id string) (p2plab.Benchmark, error) {
	return nil, nil
}

func (sapi *benchmarkAPI) List(ctx context.Context) ([]p2plab.Benchmark, error) {
	return nil, nil
}
