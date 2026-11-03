// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package selfmonitoring

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/integrationtests"
	imagemock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/image"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestSelfMonitoring(t *testing.T) {
	clt := integrationtests.SetupTestEnvironment(t)

	ctx := t.Context()

	dk := newTestDynaKube()

	integrationtests.CreateNamespace(t, clt, testNamespace)
	integrationtests.CreateDynakube(t, clt, dk)
	integrationtests.CreateKubernetesObject(t, clt, newTokenSecret(true))

	reconciler := NewReconciler(clt, clt)
	err := reconciler.Reconcile(ctx, imagemock.NewClient(t), dk)
	require.NoError(t, err)

	configHash := getObject(t, clt, &appsv1.Deployment{}).Spec.Template.Annotations[configHashAnnotation]
	assert.NotContains(t, getObject(t, clt, &corev1.ConfigMap{}).Data[configKey], "region")

	// reconcile again to exercise the update path
	dk.Spec.ResourceAttributes = map[string]string{"region": "us-east"}
	err = reconciler.Reconcile(ctx, imagemock.NewClient(t), dk)
	require.NoError(t, err)

	assert.Contains(t, getObject(t, clt, &corev1.ConfigMap{}).Data[configKey], "us-east")
	assert.NotEqual(t, configHash, getObject(t, clt, &appsv1.Deployment{}).Spec.Template.Annotations[configHashAnnotation])
}
