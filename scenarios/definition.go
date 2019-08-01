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

package scenarios

import (
	"encoding/json"
	"io/ioutil"

	"github.com/Netflix/p2plab/metadata"
)

func Parse(filename string) (metadata.ScenarioDefinition, error) {
	var sdef metadata.ScenarioDefinition
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
