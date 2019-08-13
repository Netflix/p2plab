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

package metadata

import (
	"context"
	"path/filepath"

	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
)

const (
	schemaVersion = "v1"
)

type transactionKey struct{}

func WithTransactionContext(ctx context.Context, tx *bolt.Tx) context.Context {
	return context.WithValue(ctx, transactionKey{}, tx)
}

type DB struct {
	db *bolt.DB
}

func NewDB(root string) (*DB, error) {
	path := filepath.Join(root, "meta.db")
	db, err := bolt.Open(path, 0644, nil)
	if err != nil {
		return nil, err
	}

	return &DB{db}, nil
}

func (m *DB) View(ctx context.Context, fn func(*bolt.Tx) error) error {
	tx, ok := ctx.Value(transactionKey{}).(*bolt.Tx)
	if !ok {
		return m.db.View(fn)
	}
	return fn(tx)
}

func (m *DB) Update(ctx context.Context, fn func(*bolt.Tx) error) error {
	tx, ok := ctx.Value(transactionKey{}).(*bolt.Tx)
	if !ok {
		return m.db.Update(fn)
	} else if !tx.Writable() {
		return errors.Wrap(bolt.ErrTxNotWritable, "unable to use transaction from context")
	}
	return fn(tx)
}

type field struct {
	key   []byte
	value []byte
}
