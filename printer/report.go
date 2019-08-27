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
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/Netflix/p2plab/metadata"
	"github.com/alecthomas/template"
	humanize "github.com/dustin/go-humanize"
	"github.com/hako/durafmt"
	"github.com/olekukonko/tablewriter"
)

var (
	ReportTemplate = template.Must(template.New("report").Parse(`# Summary
Total time: {{.TotalTime}}
Trace: {{.Trace}}

# Bandwidth
{{.BandwidthTable}}
# Bitswap
{{.BitswapTable}}`))
)

type ReportData struct {
	TotalTime      string
	Trace          string
	BandwidthTable string
	BitswapTable   string
}

func printReport(report metadata.Report) error {
	bwTable := printReportBandwidth(report)
	bswapTable := printReportBitswap(report)

	data := ReportData{
		TotalTime:      durafmt.Parse(report.Summary.TotalTime).String(),
		Trace:          report.Summary.Trace,
		BandwidthTable: bwTable,
		BitswapTable:   bswapTable,
	}

	err := ReportTemplate.Execute(os.Stdout, &data)
	if err != nil {
		return err
	}

	return nil
}

func printReportBandwidth(report metadata.Report) string {
	buf := new(bytes.Buffer)
	table := tablewriter.NewWriter(buf)
	table.SetAlignment(tablewriter.ALIGN_CENTER)
	table.SetHeader([]string{"Query", "Node", "TotalIn", "TotalOut", "RateIn", "RateOut"})

	qryBuckets, nodeIdsByQryBucket := sortQueryBuckets(report.Queries)
	for _, qryBucket := range qryBuckets {
		for _, nodeId := range nodeIdsByQryBucket[qryBucket] {
			totals := report.Nodes[nodeId].Bandwidth.Totals
			table.Append([]string{
				qryBucket,
				nodeId,
				humanize.Bytes(uint64(totals.TotalIn)),
				humanize.Bytes(uint64(totals.TotalOut)),
				fmt.Sprintf("%s/s", humanize.Bytes(uint64(totals.RateIn))),
				fmt.Sprintf("%s/s", humanize.Bytes(uint64(totals.RateOut))),
			})
		}
	}

	totals := report.Aggregates.Totals.Bandwidth.Totals
	table.SetFooter([]string{
		"TOTAL",
		humanize.Bytes(uint64(totals.TotalIn)),
		humanize.Bytes(uint64(totals.TotalOut)),
		fmt.Sprintf("%s/s", humanize.Bytes(uint64(totals.RateIn))),
		fmt.Sprintf("%s/s", humanize.Bytes(uint64(totals.RateOut))),
	})

	table.Render()
	return buf.String()
}

func printReportBitswap(report metadata.Report) string {
	buf := new(bytes.Buffer)
	table := tablewriter.NewWriter(buf)
	table.SetAlignment(tablewriter.ALIGN_CENTER)

	table.SetHeader([]string{"Query", "Node", "BlocksRecv", "BlocksSent", "DupBlocks", "DataRecv", "DataSent", "DupData"})

	qryBuckets, nodeIdsByQryBucket := sortQueryBuckets(report.Queries)
	for _, qryBucket := range qryBuckets {
		for _, nodeId := range nodeIdsByQryBucket[qryBucket] {
			bswap := report.Nodes[nodeId].Bitswap
			table.Append([]string{
				qryBucket,
				nodeId,
				humanize.Comma(int64(bswap.BlocksReceived)),
				humanize.Comma(int64(bswap.BlocksSent)),
				humanize.Comma(int64(bswap.DupBlksReceived)),
				humanize.Bytes(bswap.DataReceived),
				humanize.Bytes(bswap.DataSent),
				humanize.Bytes(bswap.DupDataReceived),
			})
		}
	}

	bswap := report.Aggregates.Totals.Bitswap
	table.SetFooter([]string{
		"TOTAL",
		humanize.Comma(int64(bswap.BlocksReceived)),
		humanize.Comma(int64(bswap.BlocksSent)),
		humanize.Comma(int64(bswap.DupBlksReceived)),
		humanize.Bytes(bswap.DataReceived),
		humanize.Bytes(bswap.DataSent),
		humanize.Bytes(bswap.DupDataReceived),
	})

	table.Render()
	return buf.String()
}

func sortQueryBuckets(queries map[string][]string) (qryBuckets []string, nodeIdsByQryBucket map[string][]string) {
	queriesByNodeId := make(map[string][]string)
	for qry, ns := range queries {
		for _, n := range ns {
			queriesByNodeId[n] = append(queriesByNodeId[n], qry)
		}
	}

	nodeIdsByQryBucket = make(map[string][]string)
	for nodeId, queries := range queriesByNodeId {
		var qryBucket string
		if len(queries) == 1 {
			qryBucket = queries[0]
		} else {
			sort.Strings(queries)
			qryBucket = fmt.Sprintf("(and %s)", strings.Join(queries, " "))
		}
		qryBuckets = append(qryBuckets, qryBucket)
		nodeIdsByQryBucket[qryBucket] = append(nodeIdsByQryBucket[qryBucket], nodeId)
	}
	sort.Strings(qryBuckets)

	return qryBuckets, nodeIdsByQryBucket
}
