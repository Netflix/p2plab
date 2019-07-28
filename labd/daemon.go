package labd

import (
	"context"
)

type Labd struct{}

func New() (*Labd, error) {
	return &Labd{}, nil
}

func (d *Labd) Serve(ctx context.Context) error {
	return nil
}
