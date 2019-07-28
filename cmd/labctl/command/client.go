package command

import (
	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/labd"
	"github.com/urfave/cli"
)

func ResolveClient(c *cli.Context) (p2plab.LabdAPI, error) {
	cln, err := labd.NewClient()
	if err != nil {
		return nil, err
	}

	return cln, nil
}
