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
	clocktesting "k8s.io/utils/clock/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Integration tests for the authtoken reconciler against a real API server. Drive one DynaKube
// through ordered, state-sharing phases; branch and error logic is covered by the unit test.
//
// The apiserver stamps creationTimestamp and ignores client-supplied values, so rotation can't be
// triggered by backdating the secret. Instead the reconciler runs on a fake clock the rotate phase
// advances past the production DefaultRotationInterval — no sleep.

const (
	integrationNamespace    = "dynatrace"
	integrationDynaKubeName = "lifecycle"
	integrationAPIURL       = "https://tenant.live.dynatrace.com/api"

	integrationTokenID        = "id"
	integrationInitialToken   = "initial-token"
	integrationRotatedToken   = "rotated-token"
	integrationReEnabledToken = "re-enabled-token"
)

var anyContext = mock.MatchedBy(func(context.Context) bool { return true })

type lifecycleDeps struct {
	clt        client.Client
	reconciler *authtoken.Reconciler
	clock      *clocktesting.FakePassiveClock
	dk         *dynakube.DynaKube
}

// TestReconcileLifecycle walks the phases in order: provision → stabilize → rotate → disable → re-enable.
func TestReconcileLifecycle(t *testing.T) {
	clt := integrationtests.SetupTestEnvironment(t)
	// Seed from the wall clock so it shares an epoch with the apiserver-stamped creationTimestamp.
	fakeClock := clocktesting.NewFakePassiveClock(time.Now())
	reconciler := authtoken.NewReconciler(clt, fakeClock)

	integrationtests.CreateNamespace(t, clt, integrationNamespace)

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
	integrationtests.CreateDynakube(t, clt, dk)

	deps := lifecycleDeps{
		clt:        clt,
		reconciler: reconciler,
		clock:      fakeClock,
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

	// Advance past this secret's rotation deadline, anchored to its server-stamped creationTimestamp.
	deps.clock.SetTime(oldSecret.CreationTimestamp.Add(authtoken.DefaultRotationInterval + time.Second))

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
