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
	"sort"
	"sync"
	"time"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/pkg/logutil"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

func Update(ctx context.Context, builder p2plab.Builder, ns []p2plab.Node) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "cluster healthy")
	defer span.Finish()
	span.SetTag("nodes", len(ns))

	commitByRef, err := ResolveUniqueCommits(ctx, builder, ns)
	if err != nil {
		return err
	}

	commitSet := make(map[string]struct{})
	for _, commit := range commitByRef {
		commitSet[commit] = struct{}{}
	}

	var commits []string
	for commit := range commitSet {
		commits = append(commits, commit)
	}
	sort.Strings(commits)

	linkByCommit, err := BuildCommits(ctx, builder, commits)
	if err != nil {
		return err
	}

	updatePeers, gctx := errgroup.WithContext(ctx)

	zerolog.Ctx(ctx).Info().Msg("Updating cluster")
	go logutil.Elapsed(gctx, 20*time.Second, "Updating cluster")

	for _, n := range ns {
		n := n
		updatePeers.Go(func() error {
			link := linkByCommit[commitByRef[n.Metadata().GitReference]]
			return n.Update(gctx, link)
		})
	}

	err = updatePeers.Wait()
	if err != nil {
		return err
	}

	return nil
}

func ResolveUniqueCommits(ctx context.Context, builder p2plab.Builder, ns []p2plab.Node) (commitByRef map[string]string, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "cluster healthy")
	defer span.Finish()
	span.SetTag("nodes", len(ns))

	resolving, gctx := errgroup.WithContext(ctx)

	zerolog.Ctx(ctx).Info().Msg("Resolving git references")
	go logutil.Elapsed(gctx, 20*time.Second, "Resolving git references")

	var mu sync.Mutex
	commitByRef = make(map[string]string)
	for _, n := range ns {
		n := n
		resolving.Go(func() error {
			commit, err := builder.Resolve(ctx, n.Metadata().GitReference)
			if err != nil {
				return err
			}

			mu.Lock()
			commitByRef[n.Metadata().GitReference] = commit
			mu.Unlock()
			return nil
		})
	}

	err = resolving.Wait()
	if err != nil {
		return nil, err
	}

	zerolog.Ctx(ctx).Debug().Int("references", len(commitByRef)).Msg("Resolved unique references")
	return commitByRef, nil
}

func BuildCommits(ctx context.Context, builder p2plab.Builder, commits []string) (linkByCommit map[string]string, err error) {
	logger := zerolog.Ctx(ctx).With().Strs("commits", commits).Logger()
	ctx = logger.WithContext(ctx)
	building, gctx := errgroup.WithContext(ctx)

	zerolog.Ctx(ctx).Info().Msg("Building p2p app(s)")
	go logutil.Elapsed(gctx, 20*time.Second, "Building p2p app(s)")

	var mu sync.Mutex
	linkByCommit = make(map[string]string)
	for _, commit := range commits {
		commit := commit
		building.Go(func() error {
			link, err := builder.Build(ctx, commit)
			if err != nil {
				return err
			}

			mu.Lock()
			linkByCommit[commit] = link
			mu.Unlock()
			return nil
		})
	}

	err = building.Wait()
	if err != nil {
		return nil, err
	}

	zerolog.Ctx(ctx).Debug().Int("links", len(linkByCommit)).Msg("Built unique links")
	return linkByCommit, nil
}
