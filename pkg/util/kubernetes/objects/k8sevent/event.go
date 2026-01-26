package k8sevent

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	crdVersionMismatchReason = "CRDVersionMismatch"
	crdVersionMismatchNote   = "The CustomResourceDefinition doesn't match version with the operator"
	crdVersionMismatchAction = "Please update the CRD to avoid potential issues"
)

func SendCRDVersionMismatch(eventRecorder events.EventRecorder, object client.Object) {
	eventRecorder.Eventf(
		object,
		nil,
		corev1.EventTypeWarning,
		crdVersionMismatchReason,
		crdVersionMismatchAction,
		crdVersionMismatchNote)
}
