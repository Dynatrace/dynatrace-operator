package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
)

const (
	errorInvalidServiceName = `The EdgeConnect's specification has an invalid serviceAccountName.
`
)

func isInvalidServiceName(_ context.Context, _ *validatorClient, edgeConnectCR *edgeconnect.EdgeConnect) string {
	if edgeConnectCR.GetServiceAccountName() == "" {
		return errorInvalidServiceName
	}

	return ""
}
