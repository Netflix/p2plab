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

package transformers

import (
	"net/http"
	"path/filepath"
	"sync"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/transformers/oci"
	"github.com/pkg/errors"
)

type Transformers struct {
	root   string
	client *http.Client
	mu     sync.Mutex
	ts     map[string]p2plab.Transformer
}

func New(root string, client *http.Client) *Transformers {
	return &Transformers{
		root:   root,
		client: client,
		ts:     make(map[string]p2plab.Transformer),
	}
}

func (t *Transformers) Close() error {
	for _, transformer := range t.ts {
		err := transformer.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *Transformers) Get(objectType string) (p2plab.Transformer, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	transformer, ok := t.ts[objectType]
	if !ok {
		var err error
		transformer, err = t.newTransformer(objectType)
		if err != nil {
			return nil, err
		}
		t.ts[objectType] = transformer
	}
	return transformer, nil
}

func (t *Transformers) newTransformer(objectType string) (p2plab.Transformer, error) {
	root := filepath.Join(t.root, objectType)
	switch objectType {
	case "oci":
		return oci.New(root, t.client)
	default:
		return nil, errors.Errorf("unrecognized object type: %q", objectType)
	}
}
