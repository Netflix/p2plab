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

package reports

import "github.com/Netflix/p2plab/metadata"

type uint64Pair struct {
	single    uint64
	aggregate *uint64
}

type int64Pair struct {
	single    int64
	aggregate *int64
}

type float64Pair struct {
	single    float64
	aggregate *float64
}

func ComputeAggregates(reportByNodeId map[string]metadata.ReportNode) metadata.ReportAggregates {
	var aggregates metadata.ReportAggregates
	for _, reportNode := range reportByNodeId {
		bswap := reportNode.Bitswap

		for _, pair := range []uint64Pair{
			{bswap.BlocksReceived, &aggregates.Totals.Bitswap.BlocksReceived},
			{bswap.DataReceived, &aggregates.Totals.Bitswap.DataReceived},
			{bswap.BlocksSent, &aggregates.Totals.Bitswap.BlocksSent},
			{bswap.DataSent, &aggregates.Totals.Bitswap.DataSent},
			{bswap.DupBlksReceived, &aggregates.Totals.Bitswap.DupBlksReceived},
			{bswap.DupDataReceived, &aggregates.Totals.Bitswap.DupDataReceived},
			{bswap.MessagesReceived, &aggregates.Totals.Bitswap.MessagesReceived},
		} {
			*pair.aggregate += pair.single
		}

		bandwidth := reportNode.Bandwidth.Totals
		for _, pair := range []int64Pair{
			{bandwidth.TotalIn, &aggregates.Totals.Bandwidth.Totals.TotalIn},
			{bandwidth.TotalOut, &aggregates.Totals.Bandwidth.Totals.TotalOut},
		} {
			*pair.aggregate += pair.single
		}

		for _, pair := range []float64Pair{
			{bandwidth.RateIn, &aggregates.Totals.Bandwidth.Totals.RateIn},
			{bandwidth.RateOut, &aggregates.Totals.Bandwidth.Totals.RateOut},
		} {
			*pair.aggregate += pair.single
		}
	}
	return aggregates
}
