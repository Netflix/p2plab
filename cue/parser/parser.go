package parser

import (
	"cuelang.org/go/cue"
)

var (
	// CueTemplate contains the bare cue source template used to generate
	// cue files
	CueTemplate = `
	// defines a set of nodes of size 1 or higher
	// a "node" is simply an EC2 instance provisioned of the given type
	// and there may be more than 1 node in a group, however there must always be 1
	Group :: {
		// must be greater than or equal to 1
		// default value of this field is 1
		size: >=1 | *1
		instanceType: string
		region: string
		// labels is an optional field
		labels?: [...string]
		// although not optional if left unspecified
		// then we use the default values of Peer
		peer: Peer | *Peer
	}

	Peer :: {
		gitReference: string | *"HEAD"
		transports: [...string] | *["tcp"]
		muxers: [...string] | *["mplex"]
		securityTransports: [...string] | *["secio"]
		routing: string | *"nil"
	}
	
	// a cluster is a collection of 1 or more groups of nodes
	// that will be participating in a given benchmark
	Cluster :: {
		groups: [...Group]
	}
	
	// an object is a particular data format to be used in benchmarking
	// typically these are container images
	object :: [Name=_]: { 
		type: string
		source: string
	}
	
	Scenario :: {
		objects: [...object]
		seed: { ... }
		// enable any fields for benchmark
		benchmark:  { ... }
	}
	
	Trial :: {
		cluster: Cluster
		scenario: Scenario
	}
	
	Experiment :: {
		trials: [...Trial]
	}
`
)

// Parser bundles the cue runtime with helper functions
// to enable parsing of cue source files
type Parser struct {
	entrypoints []string
	runtime     *cue.Runtime
}

// NewParser returns a ready to use cue parser
func NewParser(entrypoints []string) *Parser {
	p := &Parser{
		// entrypoints are the actual cue source files
		// we want to use when building p2plab cue files.
		// essentially they are the actual definitions
		// used to validate incoming cue source files
		entrypoints: entrypoints,
		runtime:     new(cue.Runtime),
	}
	return p
}

// Compile is used to compile the given cue source into our runtime
// it returns a wrapped cue.Instance that provides helper lookup functions
func (p *Parser) Compile(name string, cueSource string) (*P2PLabInstance, error) {
	// this is a temporary work around
	// until we can properly figure out the cue api
	for _, point := range p.entrypoints {
		cueSource += point
	}
	inst, err := p.runtime.Compile(name, cueSource)
	if err != nil {
		return nil, err
	}
	return &P2PLabInstance{inst}, nil
}
