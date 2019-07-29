package p2plab

import "context"

type ScenarioAPI interface {
	Create(ctx context.Context, name string, sdef ScenarioDefinition) (Scenario, error)

	Get(ctx context.Context, name string) (Scenario, error)

	List(ctx context.Context) ([]Scenario, error)
}

type Scenario interface {
	Remove(ctx context.Context) error
}

type ScenarioDefinition struct {
	Objects   map[string]ObjectDefinition `json:"objects,omitempty"`
	Seed      map[string]string           `json:"seed,omitempty"`
	Benchmark map[string]string           `json:"benchmark,omitempty"`
}

type ObjectDefinition struct {
	Type    string `json:"type"`
	Chunker string `json:"chunker"`
	Layout  string `json:"layout"`
}

