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
	"github.com/Netflix/p2plab/errdefs"
	"github.com/pkg/errors"
)

type Printer interface {
	Print(v interface{}) error
}

type OutputType string

var (
	OutputAuto  OutputType = "auto"
	OutputTable OutputType = "table"
	OutputID    OutputType = "id"
	OutputUnix  OutputType = "unix"
	OutputJSON  OutputType = "json"
)

func GetPrinter(output, auto OutputType) (Printer, error) {
	var p Printer
	switch output {
	case OutputAuto:
		if auto == OutputAuto {
			return nil, errors.Wrap(errdefs.ErrInvalidArgument, "auto printer cannot be auto")
		}
		return GetPrinter(auto, "")
	case OutputTable:
		p = NewTablePrinter()
	case OutputID:
		p = NewIDPrinter()
	case OutputUnix:
		p = NewUnixPrinter()
	case OutputJSON:
		p = NewJSONPrinter()
	default:
		return nil, errors.Wrapf(errdefs.ErrInvalidArgument, "output %q is not valid", output)
	}
	return p, nil
}
