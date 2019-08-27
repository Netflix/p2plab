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

type bitswapPair struct {
	single    uint64
	aggregate *uint64
}

func ComputeAggregates(reportByNodeId map[string]metadata.ReportNode) metadata.ReportAggregates {
	var aggregates metadata.ReportAggregates
	for _, reportNode := range reportByNodeId {
		singleBitswap := reportNode.Bitswap
		aggregatesBitswap := aggregates.Totals.Bitswap

		for _, pair := range []bitswapPair{
			{singleBitswap.BlocksReceived, &aggregatesBitswap.BlocksReceived},
			{singleBitswap.DataReceived, &aggregatesBitswap.DataReceived},
			{singleBitswap.BlocksSent, &aggregatesBitswap.BlocksSent},
			{singleBitswap.DataSent, &aggregatesBitswap.DataSent},
			{singleBitswap.DupBlksReceived, &aggregatesBitswap.DupBlksReceived},
			{singleBitswap.DupDataReceived, &aggregatesBitswap.DupDataReceived},
			{singleBitswap.MessagesReceived, &aggregatesBitswap.MessagesReceived},
		} {
			*pair.aggregate += pair.single
		}
	}
	return aggregates
}
