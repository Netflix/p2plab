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
	"encoding/json"
	"fmt"
	"net/http"
)

type ErrorHandler struct {
	Handler func(w http.ResponseWriter, r *http.Request) error
}

type HTTPError interface {
	error
	Status() int
}

type StatusError struct {
	Code int
	Err  error
}

func (se StatusError) Error() string {
	return se.Err.Error()
}

func (se StatusError) Status() int {
	return se.Code
}

// ServeHTTP allows our Handler type to satisfy http.Handler.
func (h ErrorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := h.Handler(w, r)
	if err != nil {
		switch e := err.(type) {
		case HTTPError:
			// We can retrieve the status here and write out a specific HTTP status code.
			fmt.Printf("HTTP %d - %s\n", e.Status(), e)
			http.Error(w, e.Error(), e.Status())
		default:
			// Any error types we don't specifically look out for default to serving a
			// HTTP 500.
			http.Error(w, e.Error(), http.StatusInternalServerError)
		}
	}
}

func WriteJSON(w http.ResponseWriter, v interface{}) error {
	content, err := json.MarshalIndent(v, "", "    ")
	if err != nil {
		return err
	}
	w.Write(content)
	return nil
}
