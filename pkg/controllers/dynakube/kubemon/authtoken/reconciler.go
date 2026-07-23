// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package authtoken

import (
	"context"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	agclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const day = 24 * time.Hour

const (
	// SecretKey is the key used inside the auth token Secret, matching the AG convention.
	SecretKey = "auth-token"

	// DefaultRotationInterval mirrors the legacy unified AG: rotate at 29 days against a 60-day token
	// validity to avoid expiry warnings in the Dynatrace UI.
	DefaultRotationInterval = 29 * day
)

type Reconciler struct {
	secrets          k8ssecret.QueryObject
	rotationInterval time.Duration
}

// NewReconciler creates a Reconciler. rotationInterval controls how long an auth token secret is
// kept before being rotated; pass DefaultRotationInterval in production, or a short-lived value
// in tests instead of waiting out the production value.
func NewReconciler(kubeClient client.Client, rotationInterval time.Duration) *Reconciler {
	return &Reconciler{
		secrets:          k8ssecret.Query(kubeClient, kubeClient),
		rotationInterval: rotationInterval,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, agClient agclient.Client, dk *dynakube.DynaKube) error {
	ctx, log := logd.NewFromContext(ctx, "kubemon-authtoken")

	if !dk.KubernetesMonitoring().IsEnabled() {
		return r.cleanup(ctx, dk)
	}

	secret, err := r.secrets.Get(ctx, client.ObjectKey{Name: dk.KubernetesMonitoring().GetAuthTokenSecretName(), Namespace: dk.Namespace})
	if err != nil && !k8serrors.IsNotFound(err) {
		return errors.WithStack(err)
	}

	if k8serrors.IsNotFound(err) {
		return r.createOrUpdateSecret(ctx, agClient, dk)
	}

	if r.isOutdated(secret) {
		log.Info("kubemon auth token is outdated, rotating", "secretName", dk.KubernetesMonitoring().GetAuthTokenSecretName())

		// Delete the old secret, so we can use creation timestamp to determine if the new secret is outdated in the next reconciliation.
		if err := r.secrets.Delete(ctx, secret); err != nil {
			return errors.WithStack(err)
		}

		return r.createOrUpdateSecret(ctx, agClient, dk)
	}

	return nil
}

func (r *Reconciler) createOrUpdateSecret(ctx context.Context, agClient agclient.Client, dk *dynakube.DynaKube) error {
	authTokenInfo, err := agClient.GetAuthToken(ctx, dk.Name)
	if err != nil {
		return errors.WithStack(err)
	}

	coreLabels := k8slabel.NewCoreLabels(dk.Name, k8slabel.KubeMonComponentLabel)

	secret, err := k8ssecret.Build(dk,
		dk.KubernetesMonitoring().GetAuthTokenSecretName(),
		map[string][]byte{SecretKey: []byte(authTokenInfo.Token)},
		k8ssecret.SetLabels(coreLabels.BuildLabels()),
	)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = r.secrets.CreateOrUpdate(ctx, secret)

	return errors.WithStack(err)
}

func (r *Reconciler) cleanup(ctx context.Context, dk *dynakube.DynaKube) error {
	return r.secrets.Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: dk.KubernetesMonitoring().GetAuthTokenSecretName(), Namespace: dk.Namespace}})
}

func (r *Reconciler) isOutdated(secret *corev1.Secret) bool {
	return secret.CreationTimestamp.Add(r.rotationInterval).Before(time.Now())
}
