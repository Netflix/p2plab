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
	"path/filepath"

	bolt "go.etcd.io/bbolt"
)

const (
	schemaVersion = "v1"
)

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

func (m *DB) View(fn func(*bolt.Tx) error) error {
	return m.db.View(fn)
}

func (m *DB) Update(fn func(*bolt.Tx) error) error {
	return m.db.Update(fn)
}

type field struct {
	key   []byte
	value []byte
}
