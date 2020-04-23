package parser

import (
	"fmt"
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
	edef, err := pinst.ToExperimentDefinition()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(edef)
}
