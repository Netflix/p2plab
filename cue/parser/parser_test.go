package parser

import (
	"io/ioutil"
	"testing"
)

func TestParser(t *testing.T) {
	data, err := ioutil.ReadFile("../cue.mod/p2plab.cue")
	if err != nil {
		t.Fatal(err)
	}
	parser := NewParser([]string{string(data)})
	data, err = ioutil.ReadFile("../cue.mod/p2plab_example.cue")
	if err != nil {
		t.Fatal(err)
	}
	pinst, err := parser.Compile("p2plab_example", string(data))
	if err != nil {
		t.Fatal(err)
	}
	_, err = pinst.ToExperimentDefinition()
	if err != nil {
		t.Fatal(err)
	}
	// manually test pinst functions that arent
	// used as  part of the ToExperimentDefinition function
	val := pinst.GetGroups()
	if val.Err() != nil {
		t.Fatal(err)
	}
	val = pinst.GetScenario()
	if val.Err() != nil {
		t.Fatal(err)
	}
	val = pinst.GetObjects()
	if val.Err() != nil {
		t.Fatal(err)
	}
}
