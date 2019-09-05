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
	"os"
	"strconv"
	"strings"

	"github.com/Netflix/p2plab/metadata"
	humanize "github.com/dustin/go-humanize"
	"github.com/olekukonko/tablewriter"
)

type tablePrinter struct{}

func NewTablePrinter() Printer {
	return &tablePrinter{}
}

func (p *tablePrinter) Print(v interface{}) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoFormatHeaders(false)

	switch t := v.(type) {
	case []interface{}:
		if len(t) > 0 {
			p.addHeader(table, t[0])
		} else {
			fmt.Println("No results")
			return nil
		}
		for _, e := range t {
			p.addRow(table, e)
		}
	case metadata.Report:
		return printReport(t)
	default:
		p.addHeader(table, t)
		p.addRow(table, t)
	}

	table.Render()
	return nil
}

func (p *tablePrinter) addHeader(table *tablewriter.Table, v interface{}) {
	switch v.(type) {
	case metadata.Cluster:
		table.SetHeader([]string{"ID", "STATUS", "SIZE", "LABELS", "CREATEDAT", "UPDATEDAT"})
	case metadata.Node:
		table.SetHeader([]string{"ID", "ADDRESS", "GITREFERENCE", "LABELS", "CREATEDAT", "UPDATEDAT"})
	case metadata.Scenario:
		table.SetHeader([]string{"ID", "LABELS", "CREATEDAT", "UPDATEDAT"})
	case metadata.Benchmark:
		table.SetHeader([]string{"ID", "STATUS", "CLUSTER", "SCENARIO", "LABELS", "CREATEDAT", "UPDATEDAT"})
	case metadata.Experiment:
		table.SetHeader([]string{"ID", "STATUS", "LABELS", "CREATEDAT", "UPDATEDAT"})
	}
}

func (p *tablePrinter) addRow(table *tablewriter.Table, v interface{}) {
	switch t := v.(type) {
	case metadata.Cluster:
		table.Append([]string{
			t.ID,
			string(t.Status),
			strconv.Itoa(t.Definition.Size()),
			strings.Join(t.Labels, ","),
			humanize.Time(t.CreatedAt),
			humanize.Time(t.UpdatedAt),
		})
	case metadata.Node:
		table.Append([]string{
			t.ID,
			t.Address,
			t.Peer.GitReference,
			strings.Join(t.Labels, ","),
			humanize.Time(t.CreatedAt),
			humanize.Time(t.UpdatedAt),
		})
	case metadata.Scenario:
		table.Append([]string{
			t.ID,
			strings.Join(t.Labels, ","),
			humanize.Time(t.CreatedAt),
			humanize.Time(t.UpdatedAt),
		})
	case metadata.Benchmark:
		table.Append([]string{
			t.ID,
			string(t.Status),
			t.Cluster.ID,
			t.Scenario.ID,
			strings.Join(t.Labels, ","),
			humanize.Time(t.CreatedAt),
			humanize.Time(t.UpdatedAt),
		})
	case metadata.Experiment:
		table.Append([]string{
			t.ID,
			string(t.Status),
			strings.Join(t.Labels, ","),
			humanize.Time(t.CreatedAt),
			humanize.Time(t.UpdatedAt),
		})
	}
}
