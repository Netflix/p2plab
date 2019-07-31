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

package printer

import (
	"fmt"

	"github.com/Netflix/p2plab/metadata"
)

type unixPrinter struct{}

func NewUnixPrinter() Printer {
	return &unixPrinter{}
}

func (p *unixPrinter) Print(v interface{}) error {
	switch t := v.(type) {
	case []interface{}:
		for _, e := range t {
			err := p.Print(e)
			if err != nil {
				return err
			}
		}
	case metadata.Cluster:
		fmt.Printf("%s\n", t.ID)
	case metadata.Node:
		fmt.Printf("%s\n", t.ID)
	case metadata.Scenario:
		fmt.Printf("%s\n", t.ID)
	case metadata.Benchmark:
		fmt.Printf("%s\n", t.ID)
	}

	return nil
}
