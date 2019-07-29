package scenario

import (
	"encoding/json"
	"io/ioutil"

	"github.com/Netflix/p2plab"
)

func Parse(filename string) (p2plab.ScenarioDefinition, error) {
	var sdef p2plab.ScenarioDefinition
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return sdef, err
	}

	err = json.Unmarshal(content, &sdef)
	if err != nil {
		return sdef, err
	}

	return sdef, nil
}
