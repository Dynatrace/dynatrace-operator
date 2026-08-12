// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sservice"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Reconciler struct {
	client   client.Client
	services k8sservice.QueryObject
}

func NewReconciler(kubeClient client.Client) *Reconciler {
	return &Reconciler{
		client:   kubeClient,
		services: k8sservice.Query(kubeClient, kubeClient),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, dk *dynakube.DynaKube) error {
	ctx, _ = logd.NewFromContext(ctx, "service")

	if !dk.KubernetesMonitoring().IsEnabled() {
		return r.services.Delete(ctx, agService(dk))
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
	svc := agService(dk)

	if err := controllerutil.SetControllerReference(dk, svc, r.client.Scheme()); err != nil {
		return err
	}

	_, err := r.services.CreateOrUpdate(ctx, svc)

	return err
}

func (r *Reconciler) setStatusIPs(ctx context.Context, dk *dynakube.DynaKube) error {
	svc := agService(dk)

	return retry.OnError(retry.DefaultBackoff, k8serrors.IsNotFound, func() error {
		currSvc, err := r.services.Get(ctx, client.ObjectKey{Name: svc.Name, Namespace: svc.Namespace})
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

func agService(dk *dynakube.DynaKube) *corev1.Service {
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

	coreLabels := k8slabel.NewCoreLabels(dk.Name, k8slabel.KubeMonComponentLabel)

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      BuildServiceName(dk.Name),
			Namespace: dk.Namespace,
			Labels:    coreLabels.BuildLabels(),
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: k8slabel.NewAppLabels(k8slabel.KubeMonComponentLabel, dk.Name, k8slabel.KubeMonComponentLabel, "").BuildMatchLabels(),
			Ports:    ports,
		},
	}
}
