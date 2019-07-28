package labagent

import (
	"context"
)

type LabAgent struct{}

func New() (*LabAgent, error) {
	return &LabAgent{}, nil
}

func (a *LabAgent) Serve(ctx context.Context) error {
	return nil
}
