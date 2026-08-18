// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package gateway

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	k8sobject "github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Reconciler struct {
	client client.Client
}

func NewReconciler(kubeClient client.Client) *Reconciler {
	return &Reconciler{
		client: kubeClient,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, dk *dynakube.DynaKube) error {
	ctx, _ = logd.NewFromContext(ctx, "gateway")

	if !dk.KubernetesMonitoring().IsEnabled() || !dk.KSPM().IsEnabled() {
		return client.IgnoreNotFound(r.client.Delete(ctx, kubemonService(dk)))
	}

	if err := r.createService(ctx, dk); err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) createService(ctx context.Context, dk *dynakube.DynaKube) error {
	desired := kubemonService(dk)
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      desired.Name,
			Namespace: desired.Namespace,
		},
	}

	_, err := k8sobject.RetryCreateOrUpdate(ctx, r.client, svc, func() error {
		svc.Labels = desired.Labels
		svc.Spec.Type = desired.Spec.Type
		svc.Spec.Selector = desired.Spec.Selector
		svc.Spec.Ports = desired.Spec.Ports

		return controllerutil.SetControllerReference(dk, svc, r.client.Scheme())
	})

	return err
}

func ServiceName(dynakubeName string) string {
	return dynakubeName + "-kubemon-activegate"
}

func kubemonService(dk *dynakube.DynaKube) *corev1.Service {
	ports := []corev1.ServicePort{
		{
			Name:       consts.HTTPSServicePortName,
			Protocol:   corev1.ProtocolTCP,
			Port:       consts.HTTPSServicePort,
			TargetPort: intstr.FromString(consts.HTTPSServicePortName),
		},
		{
			Name:       consts.HTTPServicePortName,
			Protocol:   corev1.ProtocolTCP,
			Port:       consts.HTTPServicePort,
			TargetPort: intstr.FromString(consts.HTTPServicePortName),
		},
	}

	labels := k8slabel.New(k8slabel.KubeMonComponentLabel, dk.Name, "")

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceName(dk.Name),
			Namespace: dk.Namespace,
			Labels:    labels.AsMap(),
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: labels.AsSelector(),
			Ports:    ports,
		},
	}
}
