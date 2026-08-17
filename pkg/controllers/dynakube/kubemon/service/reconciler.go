// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/util/retry"
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
	ctx, _ = logd.NewFromContext(ctx, "service")

	if !dk.KubernetesMonitoring().IsEnabled() {
		return client.IgnoreNotFound(r.client.Delete(ctx, kubemonService(dk)))
	}

	if err := r.createService(ctx, dk); err != nil {
		return err
	}

	if err := r.setStatusIPs(ctx, dk); err != nil {
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

	_, err := controllerutil.CreateOrPatch(ctx, r.client, svc, func() error {
		svc.Labels = desired.Labels
		svc.Spec.Type = desired.Spec.Type
		svc.Spec.Selector = desired.Spec.Selector
		svc.Spec.Ports = desired.Spec.Ports

		return controllerutil.SetControllerReference(dk, svc, r.client.Scheme())
	})

	return err
}

func (r *Reconciler) setStatusIPs(ctx context.Context, dk *dynakube.DynaKube) error {
	svc := kubemonService(dk)

	return retry.OnError(retry.DefaultBackoff, k8serrors.IsNotFound, func() error {
		currSvc := &corev1.Service{}

		err := r.client.Get(ctx, client.ObjectKeyFromObject(svc), currSvc)
		if err != nil {
			return err
		}

		dk.Status.KubernetesMonitoring.ServiceIPs = currSvc.Spec.ClusterIPs

		return nil
	})
}

func BuildServiceName(dynakubeName string) string {
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

	labels := k8slabel.New(k8slabel.KubeMonAppLabel, dk.Name, "")

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      BuildServiceName(dk.Name),
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
