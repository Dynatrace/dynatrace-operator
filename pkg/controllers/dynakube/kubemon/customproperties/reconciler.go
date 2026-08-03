// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0
package customproperties

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// Secret key holding the AG-internal ini.
	DataKey = "customProperties"

	// Filename expected in the pod. AG expects custom.properties in its config template directory.
	DataPath = "custom.properties"

	// Pod-spec volume name for the Secret.
	VolumeName = "kubemon-custom-properties"

	// AG expected read location.
	MountPath = "/var/lib/dynatrace/gateway/config_template/custom.properties"
)

type Reconciler struct {
	secrets k8ssecret.QueryObject
}

func NewReconciler(kubeClient client.Client) *Reconciler {
	return &Reconciler{
		secrets: k8ssecret.Query(kubeClient, kubeClient),
	}
}

// Ensures the kubemon custom-properties Secret matches Spec.CustomProperties.
// If kubemon is disabled, CustomProperties is nil, or the resolved payload is empty, the secret is deleted.
func (r *Reconciler) Reconcile(ctx context.Context, dk *dynakube.DynaKube) error {
	ctx, _ = logd.NewFromContext(ctx, "customproperties")

	if !dk.KubernetesMonitoring().IsEnabled() || dk.KubernetesMonitoring().CustomProperties == nil {
		return r.cleanup(ctx, dk)
	}

	data, err := r.resolveData(ctx, dk)
	if err != nil {
		return err
	}

	if len(data) == 0 {
		return r.cleanup(ctx, dk)
	}

	return r.ensureSecret(ctx, dk, data)
}

func (r *Reconciler) resolveData(ctx context.Context, dk *dynakube.DynaKube) ([]byte, error) {
	src := dk.KubernetesMonitoring().CustomProperties

	if src.Value != "" {
		return []byte(src.Value), nil
	}

	if src.ValueFrom == "" {
		return nil, nil
	}

	referenced, err := r.secrets.Get(ctx, client.ObjectKey{Name: src.ValueFrom, Namespace: dk.Namespace})
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to read custom properties secret %q", src.ValueFrom)
	}

	return referenced.Data[DataKey], nil
}

func (r *Reconciler) ensureSecret(ctx context.Context, dk *dynakube.DynaKube, data []byte) error {
	coreLabels := k8slabel.NewCoreLabels(dk.Name, k8slabel.KubeMonComponentLabel)

	secret, err := k8ssecret.Build(dk,
		dk.KubernetesMonitoring().GetCustomPropertiesSecretName(),
		map[string][]byte{DataKey: data},
		k8ssecret.SetLabels(coreLabels.BuildLabels()),
	)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = r.secrets.CreateOrUpdate(ctx, secret)

	return errors.WithStack(err)
}

func (r *Reconciler) cleanup(ctx context.Context, dk *dynakube.DynaKube) error {
	return r.secrets.Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
		Name:      dk.KubernetesMonitoring().GetCustomPropertiesSecretName(),
		Namespace: dk.Namespace,
	}})
}
