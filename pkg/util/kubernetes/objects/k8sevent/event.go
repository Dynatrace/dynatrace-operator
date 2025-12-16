package k8sevent

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	crdVersionMismatchReason  = "CrdVersionMismatch"
	crdVersionMismatchMessage = "The CustomResourceDefinition doesn't match version with the operator. Please update the CRD to avoid potential issues."
)

func SendCrdVersionMismatch(eventRecorder record.EventRecorder, object client.Object) {
	eventRecorder.Event(object, corev1.EventTypeWarning, crdVersionMismatchReason, crdVersionMismatchMessage)
}
