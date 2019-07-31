package labd

import (
	"net/http"

	"github.com/rs/zerolog/log"
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
			log.Info().Msgf("HTTP %d - %s", e.Status(), e)
			http.Error(w, e.Error(), e.Status())
		default:
			// Any error types we don't specifically look out for default to serving a
			// HTTP 500.
			http.Error(w, e.Error(), http.StatusInternalServerError)
		}
	}
}
