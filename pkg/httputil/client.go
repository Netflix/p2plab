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

package httputil

import (
	"net/http"
	"time"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/rs/zerolog"
)

func NewHTTPClient() *http.Client {
	return &http.Client{
		Transport: &nethttp.Transport{
			RoundTripper: cleanhttp.DefaultTransport(),
		},
	}
}

type Client struct {
	client *http.Client
	logger *zerolog.Logger
}

func NewClient(hclient *http.Client, opts ...ClientOption) (*Client, error) {
	client := &Client{
		client: hclient,
	}

	for _, opt := range opts {
		err := opt(client)
		if err != nil {
			return nil, err
		}
	}

	return client, nil
}

func (c *Client) NewRequest(method, url string, opts ...RequestOption) *Request {
	settings := RequestSettings{
		RetryWaitMin: 1 * time.Second,
		RetryWaitMax: 30 * time.Second,
		RetryMax:     4,
		CheckRetry:   retryablehttp.DefaultRetryPolicy,
		Backoff:      retryablehttp.DefaultBackoff,
	}
	for _, opt := range opts {
		opt(&settings)
	}

	client := &retryablehttp.Client{
		HTTPClient:   c.client,
		RetryWaitMin: settings.RetryWaitMin,
		RetryWaitMax: settings.RetryWaitMax,
		RetryMax:     settings.RetryMax,
		CheckRetry:   settings.CheckRetry,
		Backoff:      settings.Backoff,
	}

	if c.logger != nil {
		client.Logger = c.logger
	}

	return &Request{
		Method:  method,
		Url:     url,
		Options: make(map[string]string),
		client:  client,
	}
}

type ClientOption func(*Client) error

func WithLogger(logger *zerolog.Logger) ClientOption {
	return func(c *Client) error {
		c.logger = logger
		return nil
	}
}

type RequestOption func(*RequestSettings)

type RequestSettings struct {
	RetryWaitMin time.Duration
	RetryWaitMax time.Duration
	RetryMax     int
	CheckRetry   retryablehttp.CheckRetry
	Backoff      retryablehttp.Backoff
}

func WithRetryWaitMin(d time.Duration) RequestOption {
	return func(s *RequestSettings) {
		s.RetryWaitMin = d
	}
}

func WithRetryWaitMax(d time.Duration) RequestOption {
	return func(s *RequestSettings) {
		s.RetryWaitMax = d
	}
}

func WithRetryMax(max int) RequestOption {
	return func(s *RequestSettings) {
		s.RetryMax = max
	}
}

func WithCheckRetry(checkRetry retryablehttp.CheckRetry) RequestOption {
	return func(s *RequestSettings) {
		s.CheckRetry = checkRetry
	}
}

func WithBackoff(backoff retryablehttp.Backoff) RequestOption {
	return func(s *RequestSettings) {
		s.Backoff = backoff
	}
}
