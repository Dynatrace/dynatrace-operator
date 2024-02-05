package dynatraceapi

import (
	"net/http"

	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/pkg/errors"
)

const (
	NoError = 0
)

func IsUnreachable(err error) bool {
	var serverErr dtclient.ServerError
	if errors.As(err, &serverErr) && (serverErr.Code == http.StatusTooManyRequests || serverErr.Code == http.StatusServiceUnavailable) {
		return true
	}

	return false
}

func StatusCode(err error) int {
	var serverErr dtclient.ServerError
	if errors.As(err, &serverErr) {
		return serverErr.Code
	}

	return 0
}

func Message(err error) string {
	var serverErr dtclient.ServerError
	if errors.As(err, &serverErr) {
		return serverErr.Message
	}

	return ""
}
