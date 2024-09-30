package validation

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"golang.org/x/net/context"
)

const (
	errorInvalidServiceName = `The EdgeConnect's specification has an invalid serviceAccountName.
`
)

func isInvalidServiceName(_ context.Context, _ *Validator, edgeConnectCR *edgeconnect.EdgeConnect) string {
	if edgeConnectCR.Spec.ServiceAccountName == "" {
		return errorInvalidServiceName
	}

	return ""
}
