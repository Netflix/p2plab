package parser

import (
	"cuelang.org/go/cue"
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
