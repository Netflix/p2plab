package experiments

import (
	"encoding/csv"
	"fmt"
	"io"

	"github.com/Netflix/p2plab/metadata"
)

// ReportToCSV takes a slice of metadata reports and converts it to csv
func ReportToCSV(reports []metadata.Report, output io.Writer) error {
	w := csv.NewWriter(output)
	columns := [][]string{
		{
			"report_number",
			"total_time",
			// TODO: when testing against a large number of nodes, csv programs report errors about too much data in a single cell
			// TODO: this means we need to find a better way of representing the nodes involved in a single trial
			//	"nodes",
			"bitswap_blocks_received",
			"bitswap_data_received",
			"bitswap_blocks_sent",
			"bitswap_data_sent",
			"bitswap_dupe_blocks_received",
			"bitswap_dupe_data_received",
			"bitswap_messages_received",
			"bandwidth_total_in",
			"bandwidth_total_out",
			"bandwidth_rate_in",
			"bandwidth_rate_out",
		},
	}
	for i, report := range reports {
		/*	see todo at top of function
			var nodes string
				for node := range report.Nodes {
					nodes += fmt.Sprintf("%s%s ", nodes, node)
				}
		*/
		columns = append(columns, []string{
			fmt.Sprint(i),
			report.Summary.TotalTime.String(),
			// see todo at top of function
			// nodes,
			fmt.Sprint(report.Aggregates.Totals.Bitswap.BlocksReceived),
			fmt.Sprint(report.Aggregates.Totals.Bitswap.DataReceived),
			fmt.Sprint(report.Aggregates.Totals.Bitswap.BlocksSent),
			fmt.Sprint(report.Aggregates.Totals.Bitswap.DataSent),
			fmt.Sprint(report.Aggregates.Totals.Bitswap.DupBlksReceived),
			fmt.Sprint(report.Aggregates.Totals.Bitswap.DupDataReceived),
			fmt.Sprint(report.Aggregates.Totals.Bitswap.MessagesReceived),
			fmt.Sprint(report.Aggregates.Totals.Bandwidth.Totals.TotalIn),
			fmt.Sprint(report.Aggregates.Totals.Bandwidth.Totals.TotalOut),
			fmt.Sprint(report.Aggregates.Totals.Bandwidth.Totals.RateIn),
			fmt.Sprint(report.Aggregates.Totals.Bandwidth.Totals.RateOut),
		})
	}
	w.WriteAll(columns)
	w.Flush()
	return w.Error()
}
