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

// ToExperimentDefinition takes a cue instance and returns
// the experiment definition needed to process the experiment
func (p *P2PLabInstance) ToExperimentDefinition() (*metadata.ExperimentDefinition, error) {
	var (
		cedf metadata.ClusterDefinition
		sedf metadata.ScenarioDefinition
	)
	data, err := p.GetCluster().MarshalJSON()
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &cedf); err != nil {
		return nil, err
	}
	iter, err := p.GetObjects().List()
	if err != nil {
		return nil, err
	}
	objDef, err := getScenarioObjectDefinition(iter)
	if err != nil {
		return nil, err
	}
	sedf.Objects = objDef
	data, err = p.GetSeed().MarshalJSON()
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &sedf.Seed); err != nil {
		return nil, err
	}
	data, err = p.GetBenchmark().MarshalJSON()
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &sedf.Benchmark); err != nil {
		return nil, err
	}
	trial, err := p.TrialsToDefinition()
	if err != nil {
		return nil, err
	}
	return &metadata.ExperimentDefinition{
		ClusterDefinition:  cedf,
		ScenarioDefinition: sedf,
		TrialDefinition:    trial,
	}, nil
}

// GetExperiment returns the top-most cue value
func (p *P2PLabInstance) GetExperiment() cue.Value {
	return p.Lookup("experiment")
}

// GetCluster returns the cluster to be created as part of the benchmark
func (p *P2PLabInstance) GetCluster() cue.Value {
	return p.GetExperiment().Lookup("cluster")
}

// GetGroups returns the groups in a cluster for the given instance
func (p *P2PLabInstance) GetGroups() cue.Value {
	return p.GetCluster().Lookup("groups")
}

// GetScenario returns the scenario in an experiment fro the given instance
func (p *P2PLabInstance) GetScenario() cue.Value {
	return p.GetExperiment().Lookup("scenario")
}

// GetObjects retunrs the objects to be used in an experiment from the given instance
func (p *P2PLabInstance) GetObjects() cue.Value {
	return p.GetScenario().Lookup("objects")
}

// GetSeed returns the nodes to seed as part of the benchmark
func (p *P2PLabInstance) GetSeed() cue.Value {
	return p.GetScenario().Lookup("seed")
}

// GetBenchmark returns the benchmarks to run in p2plab
func (p *P2PLabInstance) GetBenchmark() cue.Value {
	return p.GetScenario().Lookup("benchmark")
}

// GetTrials returns the trials, a mapping of clusters and scenarios to run together
func (p *P2PLabInstance) GetTrials() cue.Value {
	return p.GetExperiment().Lookup("trials")
}

// TrialsToDefinition returns a metadata.TrialDefinition
func (p *P2PLabInstance) TrialsToDefinition() (metadata.TrialDefinition, error) {
	def := metadata.TrialDefinition{}
	val := p.GetTrials()
	if val.Err() != nil {
		return def, val.Err()
	}
	iter, err := val.List()
	if err != nil {
		return def, err
	}
	for iter.Next() {
		var (
			trial metadata.Trial
			sedf  metadata.ScenarioDefinition
			cedf  []metadata.ClusterGroup
		)
		val := iter.Value()
		if val.Err() != nil {
			return def, val.Err()
		}
		iter2, err := val.List()
		if err != nil {
			return def, err
		}
		for iter2.Next() {
			grps := iter2.Value().Lookup("groups")
			if grps.Err() == nil {
				data, err := grps.MarshalJSON()
				if err != nil {
					return def, err
				}
				if err := json.Unmarshal(data, &cedf); err != nil {
					return def, err
				}
			} else {
				// if grps.Err is not nil, then it means we have the objects definition to parse
				// because cue handles data validation, we dont need to worry about this not working
				// because if we get this far in the processing without cue throwing an error
				// this means its a likely indicator of a bug thats not a p2plab issue
				objinfo, err := iter2.Value().LookupField("objects")
				if err != nil {
					return def, err
				}
				objiter, err := objinfo.Value.List()
				if err != nil {
					return def, err
				}
				objdef, err := getScenarioObjectDefinition(objiter)
				if err != nil {
					return def, err
				}
				sedf.Objects = objdef
				seedinfo, err := iter2.Value().LookupField("seed")
				if err != nil {
					return def, err
				}
				benchinfo, err := iter2.Value().LookupField("benchmark")
				if err != nil {
					return def, err
				}
				seeddata, err := seedinfo.Value.MarshalJSON()
				if err != nil {
					return def, err
				}
				benchdata, err := benchinfo.Value.MarshalJSON()
				if err != nil {
					return def, err
				}
				if err := json.Unmarshal(seeddata, &sedf.Seed); err != nil {
					return def, err
				}
				if err := json.Unmarshal(benchdata, &sedf.Benchmark); err != nil {
					return def, err
				}
			}
		}
		trial.Cluster = cedf
		trial.Scenario = sedf
		def.Trials = append(def.Trials, trial)
	}
	return def, nil
}

func getScenarioObjectDefinition(iter cue.Iterator) (map[string]metadata.ObjectDefinition, error) {
	var (
		objData []byte
		objDef  = make(map[string]metadata.ObjectDefinition)
	)
	for iter.Next() {
		data, err := iter.Value().MarshalJSON()
		if err != nil {
			return nil, err
		}
		objData = append(objData, data...)
	}
	if err := json.Unmarshal(objData, &objDef); err != nil {
		return nil, err
	}
	return objDef, nil
}
