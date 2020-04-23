package parser

import (
	"encoding/json"

	"cuelang.org/go/cue"
	"github.com/Netflix/p2plab/metadata"
)

// P2PLabInstance is a wrapper around a cue instance
// which exposes helper functions to reduce lookup verbosity
type P2PLabInstance struct {
	*cue.Instance
}

// ToExperimentDefinition returns the cue source as a metadata.ExperimentDefinition type
func (p *P2PLabInstance) ToExperimentDefinition() (metadata.ExperimentDefinition, error) {
	trials, err := p.ToTrialDefinitions()
	if err != nil {
		return metadata.ExperimentDefinition{}, err
	}
	return metadata.ExperimentDefinition{
		TrialDefinition: trials,
	}, nil
}

// ToTrialDefinitions returns the slice of all trials to be run
func (p *P2PLabInstance) ToTrialDefinitions() ([]metadata.TrialDefinition, error) {
	var (
		def    = make([]metadata.TrialDefinition, 0)
		trials = p.Lookup("experiment").Lookup("trials")
	)
	if trials.Err() != nil {
		return nil, trials.Err()
	}
	iter, err := trials.List()
	if err != nil {
		return nil, err
	}
	for iter.Next() {
		clusterData, err := getJSON(
			iter.Value().Lookup("cluster").Lookup("groups"),
		)
		if err != nil {
			return nil, err
		}
		objIter, err := iter.Value().Lookup("scenario").Lookup("objects").List()
		if err != nil {
			return nil, err
		}
		objects, err := getScenarioObjectDefinition(objIter)
		if err != nil {
			return nil, err
		}
		seedData, err := getJSON(iter.Value().Lookup("scenario").Lookup("seed"))
		if err != nil {
			return nil, err
		}
		benchData, err := getJSON(iter.Value().Lookup("scenario").Lookup("benchmark"))
		if err != nil {
			return nil, err
		}
		trial := metadata.TrialDefinition{
			Scenario: metadata.ScenarioDefinition{
				Objects: objects,
			},
		}
		if err := json.Unmarshal(clusterData, &trial.Cluster); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(seedData, &trial.Scenario.Seed); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(benchData, &trial.Scenario.Benchmark); err != nil {
			return nil, err
		}
		def = append(def, trial)
	}
	return def, nil
}

// slightly less to type repeatedly
func getJSON(val cue.Value) ([]byte, error) {
	return val.MarshalJSON()
}

func getScenarioObjectDefinition(iter cue.Iterator) (map[string]metadata.ObjectDefinition, error) {
	objData, err := getIterData(iter)
	if err != nil {
		return nil, err
	}
	var objDef = make(map[string]metadata.ObjectDefinition)
	if err := json.Unmarshal(objData, &objDef); err != nil {
		return nil, err
	}
	return objDef, nil
}

func getIterData(iter cue.Iterator) ([]byte, error) {
	var iterData []byte
	for iter.Next() {
		data, err := getJSON(iter.Value())
		if err != nil {
			return nil, err
		}
		iterData = append(iterData, data...)
	}
	return iterData, nil
}
