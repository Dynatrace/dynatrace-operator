package k8sevent

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8scrd"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	crdVersionMismatchLabel   = "crd-name"
	crdVersionMismatchReason  = "CrdVersionMismatch"
	crdVersionMismatchMessage = "The CustomResourceDefinition %s doesn't match version with the operator. Please update the CRD to avoid potential issues."
)

var log = logd.Get().WithName("operator-k8sevent")

func SendCrdVersionMismatch(ctx context.Context, client client.Client, dk *dynakube.DynaKube, crdName string) error {
	event := corev1.Event{}

	event.Reason = crdVersionMismatchReason
	event.Message = fmt.Sprintf(crdVersionMismatchMessage, crdName)
	event.Type = corev1.EventTypeWarning
	event.Source = corev1.EventSource{Component: version.AppName}
	event.FirstTimestamp = metav1.Now()
	event.LastTimestamp = metav1.Now()
	event.Count = 1
	event.Labels = map[string]string{
		crdVersionMismatchLabel: k8scrd.DynaKubeName,
	}
	event.InvolvedObject = corev1.ObjectReference{
		APIVersion:      dk.APIVersion,
		Kind:            "customresourcedefinition",
		Namespace:       dk.Namespace,
		Name:            dk.Name,
		ResourceVersion: dk.ResourceVersion,
		UID:             dk.UID,
		FieldPath:       fmt.Sprintf("metadata.labels{%s}", k8slabel.AppVersionLabel),
	}

	event.SetGenerateName(dk.Name + "-")
	event.SetNamespace(dk.Namespace)
	event.SetGroupVersionKind(dk.GroupVersionKind())
	event.SetOwnerReferences(dk.OwnerReferences)

	log.Debug("sending k8s event %s", crdVersionMismatchReason)

	return client.Create(ctx, &event)
}
