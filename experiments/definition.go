// Copyright 2019 Netflix, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package experiments

import (
	"io/ioutil"

	"github.com/Netflix/p2plab/cue/parser"
	"github.com/Netflix/p2plab/metadata"
)

// Parse reads the cue source file at filename and converts it to a
// metadata.ExperimentDefinition type
func Parse(filename string) (metadata.ExperimentDefinition, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return metadata.ExperimentDefinition{}, err
	}
	psr := parser.NewParser([]string{parser.CueTemplate})
	inst, err := psr.Compile(
		"p2plab_experiment",
		string(content),
	)
	if err != nil {
		return metadata.ExperimentDefinition{}, err
	}
	return inst.ToExperimentDefinition()
}
