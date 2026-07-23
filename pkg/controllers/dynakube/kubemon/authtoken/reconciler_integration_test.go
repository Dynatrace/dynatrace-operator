// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package authtoken_test

import (
	"context"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	kubemonapi "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kubemon"
	agclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kubemon/authtoken"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/test/integrationtests"
	agclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/activegate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Integration tests for the authtoken reconciler against a real API server. Drive one DynaKube
// through ordered, state-sharing phases; branch and error logic is covered by the unit test.
//
// Rotation is driven by comparing the server-managed creationTimestamp against the current time,
// and a real apiserver sets that field on create and ignores client-supplied values on update, so
// it cannot be backdated through the client. Instead, the rotate phase configures a short
// rotationInterval via authtoken.WithRotationInterval and waits out real elapsed time.

const (
	integrationNamespace    = "dynatrace"
	integrationDynaKubeName = "lifecycle"
	integrationAPIURL       = "https://tenant.live.dynatrace.com/api"

	integrationTokenID        = "id"
	integrationInitialToken   = "initial-token"
	integrationRotatedToken   = "rotated-token"
	integrationReEnabledToken = "re-enabled-token"

	// integrationRotationInterval must stay comfortably longer than a couple of back-to-back
	// reconciles (see the stabilize phase) so it doesn't trigger a spurious rotation there, while
	// staying short enough that the rotate phase can wait it out on real elapsed time.
	integrationRotationInterval = time.Second
)

var anyContext = mock.MatchedBy(func(context.Context) bool { return true })

type lifecycleDeps struct {
	clt        client.Client
	reconciler *authtoken.Reconciler
	dk         *dynakube.DynaKube
}

// TestReconcileLifecycle walks the phases in order: provision → stabilize → rotate → disable → re-enable.
func TestReconcileLifecycle(t *testing.T) {
	clt := integrationtests.SetupTestEnvironment(t)
	ctx := t.Context()
	reconciler := authtoken.NewReconciler(clt, authtoken.WithRotationInterval(integrationRotationInterval))

	integrationtests.CreateNamespace(t, ctx, clt, integrationNamespace)

	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      integrationDynaKubeName,
			Namespace: integrationNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL:               integrationAPIURL,
			KubernetesMonitoring: &kubemonapi.Spec{},
		},
	}
	integrationtests.CreateDynakube(t, ctx, clt, dk)

	deps := lifecycleDeps{
		clt:        clt,
		reconciler: reconciler,
		dk:         dk,
	}

	t.Run("provision", func(t *testing.T) { runProvisionPhase(t, deps) })
	t.Run("stabilize", func(t *testing.T) { runStabilizePhase(t, deps) })
	t.Run("rotate", func(t *testing.T) { runRotatePhase(t, deps) })
	t.Run("disable", func(t *testing.T) { runDisablePhase(t, deps) })
	t.Run("re-enable", func(t *testing.T) { runReEnablePhase(t, deps) })
}

func runProvisionPhase(t *testing.T, deps lifecycleDeps) {
	t.Helper()

	dtClient := agclientmock.NewClient(t)
	dtClient.EXPECT().GetAuthToken(anyContext, deps.dk.Name).Return(&agclient.AuthTokenInfo{TokenID: integrationTokenID, Token: integrationInitialToken}, nil)

	// Create path is deterministic: the reconciler's Get of a never-created secret returns NotFound,
	// so a single reconcile creates it and the read-back observes it immediately.
	require.NoError(t, deps.reconciler.Reconcile(t.Context(), dtClient, deps.dk))

	secret := getSecret(t, deps.clt, deps.dk)
	assert.Len(t, secret.Data, 1)
	assert.Equal(t, []byte(integrationInitialToken), secret.Data[authtoken.SecretKey])
	assert.Equal(t, k8slabel.KubeMonComponentLabel, secret.Labels[k8slabel.AppComponentLabel])
	assert.Equal(t, deps.dk.Name, secret.Labels[k8slabel.AppCreatedByLabel])
	assert.True(t, metav1.IsControlledBy(secret, deps.dk))
}

func runStabilizePhase(t *testing.T, deps lifecycleDeps) {
	t.Helper()

	dtClient := agclientmock.NewClient(t)
	// No GetAuthToken expectation — a stable, fresh secret must not trigger a rotation.

	secretRV := getSecret(t, deps.clt, deps.dk).ResourceVersion

	// Repeated reconciles with identical input must not rewrite the secret.
	for range 3 {
		require.NoError(t, deps.reconciler.Reconcile(t.Context(), dtClient, deps.dk))
		assert.Equal(t, secretRV, getSecret(t, deps.clt, deps.dk).ResourceVersion)
	}
}

func runRotatePhase(t *testing.T, deps lifecycleDeps) {
	t.Helper()

	dtClient := agclientmock.NewClient(t)
	dtClient.EXPECT().GetAuthToken(anyContext, deps.dk.Name).Return(&agclient.AuthTokenInfo{TokenID: integrationTokenID, Token: integrationRotatedToken}, nil)

	oldSecret := getSecret(t, deps.clt, deps.dk)

	// Wait out the real rotationInterval configured on the shared reconciler; the creationTimestamp
	// is server-managed and can't be backdated through the client. A single reconcile then observes
	// the now-outdated secret and rotates it.
	time.Sleep(integrationRotationInterval + 200*time.Millisecond)

	require.NoError(t, deps.reconciler.Reconcile(t.Context(), dtClient, deps.dk))

	rotated := getSecret(t, deps.clt, deps.dk)
	assert.Equal(t, []byte(integrationRotatedToken), rotated.Data[authtoken.SecretKey])
	assert.NotEqual(t, oldSecret.UID, rotated.UID, "rotation must delete and recreate the secret")
}

func runDisablePhase(t *testing.T, deps lifecycleDeps) {
	t.Helper()

	deps.dk.Spec.KubernetesMonitoring = nil

	require.NoError(t, deps.reconciler.Reconcile(t.Context(), nil, deps.dk))

	err := deps.clt.Get(t.Context(), types.NamespacedName{Name: deps.dk.KubernetesMonitoring().GetAuthTokenSecretName(), Namespace: deps.dk.Namespace}, &corev1.Secret{})
	require.True(t, k8serrors.IsNotFound(err), "secret should be deleted when kubemon is disabled")
}

func runReEnablePhase(t *testing.T, deps lifecycleDeps) {
	t.Helper()

	deps.dk.Spec.KubernetesMonitoring = &kubemonapi.Spec{}
	dtClient := agclientmock.NewClient(t)
	dtClient.EXPECT().GetAuthToken(anyContext, deps.dk.Name).Return(&agclient.AuthTokenInfo{TokenID: integrationTokenID, Token: integrationReEnabledToken}, nil)

	require.NoError(t, deps.reconciler.Reconcile(t.Context(), dtClient, deps.dk))
	assert.Equal(t, []byte(integrationReEnabledToken), getSecret(t, deps.clt, deps.dk).Data[authtoken.SecretKey])
}

func getSecret(t *testing.T, reader client.Reader, dk *dynakube.DynaKube) *corev1.Secret {
	t.Helper()

	secret := &corev1.Secret{}
	require.NoError(t, reader.Get(t.Context(), types.NamespacedName{Name: dk.KubernetesMonitoring().GetAuthTokenSecretName(), Namespace: dk.Namespace}, secret))

	return secret
}
