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

package daemon

import (
	"context"
	"net/http"
)

type Router interface {
	Routes() []Route
}

type Route interface {
	Method() string
	Path() string
	Handler() Handler
}

type Handler func(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error

type route struct {
	method  string
	path    string
	handler Handler
}

func NewRoute(method, path string, handler Handler) Route {
	return &route{method, path, handler}
}

func (r *route) Method() string {
	return r.method
}

func (r *route) Path() string {
	return r.path
}

func (r *route) Handler() Handler {
	return r.handler
}

func NewGetRoute(path string, handler Handler) Route {
	return NewRoute("GET", path, handler)
}

func NewPostRoute(path string, handler Handler) Route {
	return NewRoute("POST", path, handler)
}

func NewPutRoute(path string, handler Handler) Route {
	return NewRoute("PUT", path, handler)
}

func NewDeleteRoute(path string, handler Handler) Route {
	return NewRoute("DELETE", path, handler)
}

func NewOptionsRoute(path string, handler Handler) Route {
	return NewRoute("OPTIONS", path, handler)
}

func NewHeadRoute(path string, handler Handler) Route {
	return NewRoute("HEAD", path, handler)
}
