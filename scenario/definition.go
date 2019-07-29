package scenario

import (
	"encoding/json"
	"io/ioutil"
)

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

func Parse(filename string) (ScenarioDefinition, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var sdef ScenarioDefinition
	err = json.Unmarshal(content, &sdef)
	if err != nil {
		return nil, err
	}

	return sdef, nil
}
