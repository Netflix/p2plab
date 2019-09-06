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

package nodes

import (
	"context"
	"sync"
	"time"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/pkg/logutil"
	"github.com/Netflix/p2plab/pkg/traceutil"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

func CollectReports(ctx context.Context, ns []p2plab.Node) (map[string]metadata.ReportNode, error) {
	span, ctx := traceutil.StartSpanFromContext(ctx, "nodes.CollectReports")
	defer span.Finish()
	span.SetTag("nodes", len(ns))

	getReports, gctx := errgroup.WithContext(ctx)

	zerolog.Ctx(ctx).Info().Msg("Retrieving reports")
	go logutil.Elapsed(gctx, 20*time.Second, "Retrieving reports")

	var mu sync.Mutex
	reportByNodeID := make(map[string]metadata.ReportNode)

	for _, n := range ns {
		n := n
		getReports.Go(func() error {
			report, err := n.Report(ctx)
			if err != nil {
				return err
			}

			mu.Lock()
			reportByNodeID[n.ID()] = report
			mu.Unlock()
			return nil
		})
	}

	err := getReports.Wait()
	if err != nil {
		return nil, err
	}

	return reportByNodeID, nil
}
