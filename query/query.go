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

package query

import (
	"context"
	"fmt"
	"strings"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/errdefs"
	"github.com/gobwas/glob"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// query := label
//        | '(' func expr ')'
// expr := query
//       | query expr
// func := ‘not’
//       | ‘and’
//       | ‘or’
// label := quoted_string
func Parse(ctx context.Context, q string) (p2plab.Query, error) {
	tokens := tokenize(q)
	if len(tokens) == 0 {
		tokens = []string{"*"}
	}

	var (
		qry p2plab.Query
		err error
	)
	if len(tokens) == 1 {
		label := strings.Trim(tokens[0], "'")
		qry, err = newLabelQuery(fmt.Sprintf("'%s'", label))
	} else {
		qry, err = buildQuery(tokens)
	}
	if err != nil {
		return nil, errors.Wrapf(errdefs.ErrInvalidArgument, "%s", err)
	}

	zerolog.Ctx(ctx).Debug().Msgf("Parsed query as %q", qry)
	return qry, nil
}

func tokenize(q string) []string {
	q = strings.ReplaceAll(q, "(", " ( ")
	q = strings.ReplaceAll(q, ")", " ) ")

	var tokens []string
	rawTokens := strings.Split(q, " ")
	for _, t := range rawTokens {
		stripped := strings.TrimSpace(t)
		if stripped == "" {
			continue
		}
		tokens = append(tokens, stripped)
	}
	return tokens
}

func buildQuery(tokens []string) (p2plab.Query, error) {
	// First token is either a start of a function or a label query.
	if tokens[0] != "(" {
		if len(tokens) > 1 {
			return nil, errors.New("unexpected trailing tokens")
		}

		return newLabelQuery(tokens[0])
	}

	if len(tokens[1:]) < 2 {
		return nil, errors.New("query must have function and expression")
	} else if tokens[len(tokens)-1] != ")" {
		return nil, errors.New("query must end in a closing parenthesis")
	}

	queries, err := buildExpression(tokens[2 : len(tokens)-1])
	if err != nil {
		return nil, err
	}

	switch tokens[1] {
	case "not":
		return newNotQuery(queries)
	case "and":
		return newAndQuery(queries)
	case "or":
		return newOrQuery(queries)
	default:
		return nil, errors.Errorf("unrecognized function %q", tokens[1])
	}
}

func buildExpression(tokens []string) ([]p2plab.Query, error) {
	var queries []p2plab.Query

	for i := 0; i < len(tokens); {
		// If the next token is a label, then it is a single element.
		// Otherwise it is a beginning of a longer query.
		j := i + 1
		if tokens[i] == "(" {
			parens := 1
			for ; j < len(tokens) && parens > 0; j++ {
				switch tokens[j] {
				case "(":
					parens++
				case ")":
					parens--
				}
			}

			if parens != 0 {
				return nil, errors.New("no matching parenthesis")
			}
		}

		qry, err := buildQuery(tokens[i:j])
		if err != nil {
			return nil, err
		}
		queries = append(queries, qry)

		// Move i to the end of the scanned tokens.
		i = j
	}

	return queries, nil
}

type notQuery struct {
	query p2plab.Query
}

func newNotQuery(queries []p2plab.Query) (p2plab.Query, error) {
	if len(queries) != 1 {
		return nil, errors.New("not query must have exactly 1 argument")
	}
	return &notQuery{queries[0]}, nil
}

func (q *notQuery) String() string {
	return fmt.Sprintf("(not %s)", q.query)
}

func (q *notQuery) Match(ctx context.Context, lset p2plab.LabeledSet) (p2plab.LabeledSet, error) {
	positiveSet, err := q.query.Match(ctx, lset)
	if err != nil {
		return nil, err
	}

	negativeSet := NewLabeledSet()
	for _, l := range lset.Slice() {
		if positiveSet.Contains(l.ID()) {
			continue
		}
		negativeSet.Add(l)
	}

	return negativeSet, nil
}

type andQuery struct {
	queries []p2plab.Query
}

func newAndQuery(queries []p2plab.Query) (p2plab.Query, error) {
	return &andQuery{queries}, nil
}

func (q *andQuery) String() string {
	var r []string
	for _, q := range q.queries {
		r = append(r, q.String())
	}
	return fmt.Sprintf("(and %s)", strings.Join(r, " "))
}

func (q *andQuery) Match(ctx context.Context, lset p2plab.LabeledSet) (p2plab.LabeledSet, error) {
	var qsets []p2plab.LabeledSet
	for _, q := range q.queries {
		qset, err := q.Match(ctx, lset)
		if err != nil {
			return nil, err
		}
		qsets = append(qsets, qset)
	}

	andSet := NewLabeledSet()
	for _, l := range lset.Slice() {
		allContains := true
		for _, qset := range qsets {
			if !qset.Contains(l.ID()) {
				allContains = false
				break
			}
		}

		if allContains {
			andSet.Add(l)
		}
	}

	return andSet, nil
}

type orQuery struct {
	queries []p2plab.Query
}

func newOrQuery(queries []p2plab.Query) (p2plab.Query, error) {
	return &orQuery{queries}, nil
}

func (q *orQuery) String() string {
	var r []string
	for _, q := range q.queries {
		r = append(r, q.String())
	}
	return fmt.Sprintf("(or %s)", strings.Join(r, " "))
}

func (q *orQuery) Match(ctx context.Context, lset p2plab.LabeledSet) (p2plab.LabeledSet, error) {
	var qsets []p2plab.LabeledSet
	for _, q := range q.queries {
		qset, err := q.Match(ctx, lset)
		if err != nil {
			return nil, err
		}
		qsets = append(qsets, qset)
	}

	orSet := NewLabeledSet()
	for _, l := range lset.Slice() {
		oneContains := false
		for _, qset := range qsets {
			if qset.Contains(l.ID()) {
				oneContains = true
				break
			}
		}

		if oneContains {
			orSet.Add(l)
		}
	}

	return orSet, nil
}

type labelQuery struct {
	pattern string
	glob    glob.Glob
}

func newLabelQuery(label string) (p2plab.Query, error) {
	if len(label) < 3 {
		return nil, errors.Errorf("label must be at least 2 length: %q", label)
	}
	if label[0] != '\'' || label[len(label)-1] != '\'' {
		return nil, errors.Errorf("label must be in quotations: %q", label)
	}

	pattern := label[1 : len(label)-1]
	g, err := glob.Compile(pattern)
	if err != nil {
		return nil, err
	}

	return &labelQuery{
		pattern: pattern,
		glob:    g,
	}, nil
}

func (q *labelQuery) String() string {
	return fmt.Sprintf("'%s'", q.pattern)
}

func (q *labelQuery) Match(ctx context.Context, lset p2plab.LabeledSet) (p2plab.LabeledSet, error) {
	labelSet := NewLabeledSet()
	for _, l := range lset.Slice() {
		found := false
		for _, label := range l.Labels() {
			if q.glob.Match(label) {
				found = true
				break
			}
		}

		if found || (q.pattern == "*" && len(l.Labels()) == 0) {
			labelSet.Add(l)
		}
	}

	return labelSet, nil
}
