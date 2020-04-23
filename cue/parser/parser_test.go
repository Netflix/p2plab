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
	var sourceFiles = []string{
		"../cue.mod/p2plab_example1.cue",
		"../cue.mod/p2plab_example2.cue",
	}
	for _, sourceFile := range sourceFiles {
		data, err = ioutil.ReadFile(sourceFile)
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
	}

}
