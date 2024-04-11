package edgeconnect

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	"golang.org/x/net/context"
)

const (
	errorInvalidServiceName = `The EdgeConnect's specification has an invalid serviceAccountName.
`
)

func isInvalidServiceName(_ context.Context, _ *edgeconnectValidator, edgeConnectCR *edgeconnect.EdgeConnect) string {
	if edgeConnectCR.Spec.ServiceAccountName == "" {
		return errorInvalidServiceName
	}

	return ""
}
