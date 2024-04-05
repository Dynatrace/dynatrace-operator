package edgeconnect

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errorInvalidServiceName = `The EdgeConnect's specification has an invalid serviceAccountName.
`
	errorServiceAccountNotExist = `The EdgeConnect's specification has an invalid serviceAccountName. ServiceAccount doesn't exist
`
)

func isInvalidServiceName(_ context.Context, _ *edgeconnectValidator, edgeConnectCR *edgeconnect.EdgeConnect) string {
	if edgeConnectCR.Spec.ServiceAccountName == "" {
		return errorInvalidServiceName
	}

	return ""
}

func doesServiceAccountExist(ctx context.Context, validator *edgeconnectValidator, edgeConnectCR *edgeconnect.EdgeConnect) string {
	if edgeConnectCR.Spec.ServiceAccountName != "" {
		var serviceAccount corev1.ServiceAccount

		err := validator.apiReader.Get(ctx, client.ObjectKey{Name: edgeConnectCR.Spec.ServiceAccountName, Namespace: edgeConnectCR.Namespace}, &serviceAccount)
		if err != nil && k8sErrors.IsNotFound(err) {
			return errorServiceAccountNotExist
		} else if err != nil {
			log.Info("The EdgeConnect's specification can't be verified due to the following error", "error", err)
		}
	}

	return ""
}
