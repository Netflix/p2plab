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

package labd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Request struct {
	Client   *http.Client
	Method   string
	Base     string
	Endpoint string
	Options  map[string]string
	Body     io.Reader
}

func (c *client) NewRequest(method, path string, a ...interface{}) *Request {
	return &Request{
		Client:   c.httpClient,
		Method:   method,
		Base:     c.base,
		Endpoint: fmt.Sprintf(path, a...),
		Options:  make(map[string]string),
	}
}

func (r *Request) Option(k, v string) *Request {
	r.Options[k] = v
	return r
}

func (r *Request) Send(ctx context.Context) (*http.Response, error) {
	req, err := http.NewRequest(r.Method, r.url(), r.Body)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)
	resp, err := r.Client.Do(req)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

func (r *Request) url() string {
	values := make(url.Values)
	for k, v := range r.Options {
		values.Add(k, v)
	}

	return fmt.Sprintf("%s/%s?%s", r.Base, r.Endpoint, values.Encode())
}
