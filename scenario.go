package p2plab

import "context"

type ScenarioAPI interface {
	Get(ctx context.Context, id string) (Scenario, error)

	List(ctx context.Context) ([]Scenario, error)
}

type Scenario interface {
}
