package grzybek

import (
	"net/http"

	"github.com/go-logr/logr"
)

func NewHttpRequestHandler(log logr.Logger) func(r *http.Request) error {
	return func(r *http.Request) error {
		log.Info("got request:", "URI", r.RequestURI)
		return nil
	}
}
