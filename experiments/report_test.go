package experiments

import (
	"os"
	"testing"
	"time"

	metrics "github.com/libp2p/go-libp2p-core/metrics"
	"github.com/stretchr/testify/require"

	"github.com/Netflix/p2plab/metadata"
)

func TestReportToCSV(t *testing.T) {
	testFile := "report.csv"
	t.Cleanup(func() {
		require.NoError(t, os.Remove(testFile))
	})
	reports := []metadata.Report{
		{
			Summary: metadata.ReportSummary{
				TotalTime: time.Hour,
			},
			Aggregates: metadata.ReportAggregates{
				Totals: metadata.ReportNode{
					Bitswap: metadata.ReportBitswap{
						BlocksReceived:   1,
						DataReceived:     2,
						BlocksSent:       3,
						DataSent:         4,
						DupBlksReceived:  5,
						DupDataReceived:  6,
						MessagesReceived: 7,
					},
					Bandwidth: metadata.ReportBandwidth{
						Totals: metrics.Stats{},
					},
				},
			},
			Nodes: map[string]metadata.ReportNode{
				"node1": {},
				"node2": {},
				"node3": {},
			},
		},
	}
	fh, err := os.Create(testFile)
	require.NoError(t, err)
	defer require.NoError(t, fh.Close())
	require.NoError(t, ReportToCSV(reports, fh))
}
