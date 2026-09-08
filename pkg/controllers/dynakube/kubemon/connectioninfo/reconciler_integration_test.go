// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package connectioninfo_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	kubemonapi "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kubemon"
	agclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core/middleware"
	sharedconnectioninfo "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kubemon/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/test/integrationtests"
	agclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/activegate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Integration tests for the connectioninfo reconciler against a real API server. Drives one DynaKube
// through ordered, state-sharing phases; each phase asserts with a single reconcile call against a
// direct API client. Branch and error logic is covered by the unit test.

const (
	integrationNamespace    = "dynatrace"
	integrationDynaKubeName = "lifecycle"
	integrationAPIURL       = "https://tenant.live.dynatrace.com/api"
	integrationTenantUUID   = "test-uuid"
	integrationEndpoints    = "https://tenant.live.dynatrace.com/communication"
	integrationTenantToken  = "test-token"

	integrationRotatedEndpoints   = "https://tenant.live.dynatrace.com/updated"
	integrationRotatedTenantToken = "rotated-token"
)

var anyContext = mock.MatchedBy(func(context.Context) bool { return true })

type lifecycleDeps struct {
	clt                    client.Client
	reconciler             *connectioninfo.Reconciler
	dk                     *dynakube.DynaKube
	baselineConnectionInfo agclient.ConnectionInfo
	rotatedConnectionInfo  agclient.ConnectionInfo
}

// TestReconcileLifecycle walks the phases in order: provision → rotate → stabilize → disable → re-enable.
func TestReconcileLifecycle(t *testing.T) {
	clt := integrationtests.SetupTestEnvironment(t)
	reconciler := connectioninfo.NewReconciler(clt)

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

	baselineConnectionInfo := agclient.ConnectionInfo{
		TenantUUID:  integrationTenantUUID,
		TenantToken: integrationTenantToken,
		Endpoints:   integrationEndpoints,
	}

	rotatedConnectionInfo := agclient.ConnectionInfo{
		TenantUUID:  integrationTenantUUID,
		TenantToken: integrationRotatedTenantToken,
		Endpoints:   integrationRotatedEndpoints,
	}

	// The subtests below share dk and run in order: each builds on the state left by the previous one.
	deps := lifecycleDeps{
		clt:                    clt,
		reconciler:             reconciler,
		dk:                     dk,
		baselineConnectionInfo: baselineConnectionInfo,
		rotatedConnectionInfo:  rotatedConnectionInfo,
	}

	t.Run("provision", func(t *testing.T) { runProvisionPhase(t, deps) })
	t.Run("rotate", func(t *testing.T) { runRotatePhase(t, deps) })
	t.Run("stabilize", func(t *testing.T) { runStabilizePhase(t, deps) })
	t.Run("disable", func(t *testing.T) { runDisablePhase(t, deps) })
	t.Run("re-enable", func(t *testing.T) { runReEnablePhase(t, deps) })
}

func runProvisionPhase(t *testing.T, deps lifecycleDeps) {
	t.Helper()

	dtClient := agclientmock.NewClient(t)
	dtClient.EXPECT().GetConnectionInfo(anyContext).Return(deps.baselineConnectionInfo, nil)

	require.NoError(t, deps.reconciler.Reconcile(t.Context(), dtClient, deps.dk))
	require.True(t, isConnectionInfoApplied(t.Context(), deps.clt, deps.dk, deps.baselineConnectionInfo))

	cm := getConfigMap(t, deps.clt, deps.dk)
	assert.Len(t, cm.Data, 2)
	assertManagedLabels(t, cm.Labels, deps.dk)
	assert.True(t, metav1.IsControlledBy(cm, deps.dk))

	secret := getSecret(t, deps.clt, deps.dk)
	assert.Len(t, secret.Data, 1)
	assertManagedLabels(t, secret.Labels, deps.dk)
	assert.True(t, metav1.IsControlledBy(secret, deps.dk))
}

func runRotatePhase(t *testing.T, deps lifecycleDeps) {
	t.Helper()

	dtClient := agclientmock.NewClient(t)
	dtClient.EXPECT().GetConnectionInfo(anyContext).Return(deps.rotatedConnectionInfo, nil)

	require.NoError(t, deps.reconciler.Reconcile(t.Context(), dtClient, deps.dk))
	require.True(t, isConnectionInfoApplied(t.Context(), deps.clt, deps.dk, deps.rotatedConnectionInfo))
}

func runStabilizePhase(t *testing.T, deps lifecycleDeps) {
	t.Helper()

	dtClient := agclientmock.NewClient(t)
	dtClient.EXPECT().GetConnectionInfo(anyContext).Return(deps.rotatedConnectionInfo, nil)

	cmRV := getConfigMap(t, deps.clt, deps.dk).ResourceVersion
	secretRV := getSecret(t, deps.clt, deps.dk).ResourceVersion

	// Repeated reconciles with identical input must not rewrite resources.
	for range 3 {
		require.NoError(t, deps.reconciler.Reconcile(t.Context(), dtClient, deps.dk))
		assert.Equal(t, cmRV, getConfigMap(t, deps.clt, deps.dk).ResourceVersion)
		assert.Equal(t, secretRV, getSecret(t, deps.clt, deps.dk).ResourceVersion)
	}
}

func runDisablePhase(t *testing.T, deps lifecycleDeps) {
	t.Helper()

	deps.dk.Spec.KubernetesMonitoring = nil

	require.NoError(t, deps.reconciler.Reconcile(t.Context(), nil, deps.dk))

	cmErr := deps.clt.Get(t.Context(), client.ObjectKey{Name: deps.dk.KubernetesMonitoring().GetConnectionInfoConfigMapName(), Namespace: deps.dk.Namespace}, &corev1.ConfigMap{})
	secretErr := deps.clt.Get(t.Context(), client.ObjectKey{Name: deps.dk.KubernetesMonitoring().GetTenantSecretName(), Namespace: deps.dk.Namespace}, &corev1.Secret{})
	require.True(t, k8serrors.IsNotFound(cmErr))
	require.True(t, k8serrors.IsNotFound(secretErr))
	assert.Empty(t, deps.dk.Status.KubernetesMonitoring.ConnectionInfo.TenantUUID)
	assert.Empty(t, deps.dk.Status.KubernetesMonitoring.ConnectionInfo.Endpoints)
}

func runReEnablePhase(t *testing.T, deps lifecycleDeps) {
	t.Helper()

	deps.dk.Spec.KubernetesMonitoring = &kubemonapi.Spec{}
	dtClient := agclientmock.NewClient(t)
	dtClient.EXPECT().GetConnectionInfo(anyContext).Return(deps.baselineConnectionInfo, nil)

	require.NoError(t, deps.reconciler.Reconcile(t.Context(), dtClient, deps.dk))
	require.True(t, isConnectionInfoApplied(t.Context(), deps.clt, deps.dk, deps.baselineConnectionInfo))
}

func getConfigMap(t *testing.T, reader client.Reader, dk *dynakube.DynaKube) *corev1.ConfigMap {
	t.Helper()

	cm := &corev1.ConfigMap{}
	require.NoError(t, reader.Get(t.Context(), client.ObjectKey{Name: dk.KubernetesMonitoring().GetConnectionInfoConfigMapName(), Namespace: dk.Namespace}, cm))

	return cm
}

func getSecret(t *testing.T, reader client.Reader, dk *dynakube.DynaKube) *corev1.Secret {
	t.Helper()

	secret := &corev1.Secret{}
	require.NoError(t, reader.Get(t.Context(), client.ObjectKey{Name: dk.KubernetesMonitoring().GetTenantSecretName(), Namespace: dk.Namespace}, secret))

	return secret
}

func isConnectionInfoApplied(ctx context.Context, reader client.Reader, dk *dynakube.DynaKube, info agclient.ConnectionInfo) bool {
	if dk.Status.KubernetesMonitoring.ConnectionInfo.TenantUUID != info.TenantUUID ||
		dk.Status.KubernetesMonitoring.ConnectionInfo.Endpoints != info.Endpoints {
		return false
	}

	cm := &corev1.ConfigMap{}
	if err := reader.Get(ctx, client.ObjectKey{Name: dk.KubernetesMonitoring().GetConnectionInfoConfigMapName(), Namespace: dk.Namespace}, cm); err != nil {
		return false
	}

	if cm.Data[sharedconnectioninfo.TenantUUIDKey] != info.TenantUUID ||
		cm.Data[sharedconnectioninfo.CommunicationEndpointsKey] != info.Endpoints {
		return false
	}

	secret := &corev1.Secret{}
	if err := reader.Get(ctx, client.ObjectKey{Name: dk.KubernetesMonitoring().GetTenantSecretName(), Namespace: dk.Namespace}, secret); err != nil {
		return false
	}

	return string(secret.Data[sharedconnectioninfo.TenantTokenKey]) == info.TenantToken
}

func assertManagedLabels(t *testing.T, labels map[string]string, dk *dynakube.DynaKube) {
	t.Helper()

	assert.Equal(t, k8slabel.KubeMonComponentLabel, labels[k8slabel.AppComponentLabel])
	assert.Equal(t, dk.Name, labels[k8slabel.AppCreatedByLabel])
}

// TestConnectionInfoCache verifies that the HTTP-level response cache used by the AG
// client prevents redundant API calls when the reconciler is called multiple times.
// envtest has real goroutines doing I/O, so synctest is not used; the expiry subtest
// uses a short TTL with a real time.Sleep instead.
func TestConnectionInfoCache(t *testing.T) {
	clt := integrationtests.SetupTestEnvironment(t)
	reconciler := connectioninfo.NewReconciler(clt)

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

	t.Run("uses cached connection info within TTL", func(t *testing.T) {
		transport := newFakeCITransport(
			kubemonConnectionInfoBody(integrationTenantUUID, integrationTenantToken, integrationEndpoints),
		)
		agClient := newKubemonConnectionInfoClient(t, transport, time.Minute)

		require.NoError(t, reconciler.Reconcile(t.Context(), agClient, dk))
		require.NoError(t, reconciler.Reconcile(t.Context(), agClient, dk))

		transport.assertCalls(t, 1, "second reconcile must be served from cache")
	})

	t.Run("fetches fresh connection info after cache TTL expires", func(t *testing.T) {
		const shortTTL = 10 * time.Millisecond
		transport := newFakeCITransport(
			kubemonConnectionInfoBody(integrationTenantUUID, integrationTenantToken, integrationEndpoints),
			kubemonConnectionInfoBody(integrationTenantUUID, integrationTenantToken, integrationRotatedEndpoints),
		)
		agClient := newKubemonConnectionInfoClient(t, transport, shortTTL)

		require.NoError(t, reconciler.Reconcile(t.Context(), agClient, dk))

		time.Sleep(2 * shortTTL)

		require.NoError(t, reconciler.Reconcile(t.Context(), agClient, dk))

		assert.Equal(t, integrationRotatedEndpoints, dk.Status.KubernetesMonitoring.ConnectionInfo.Endpoints)
		transport.assertCalls(t, 2)
	})
}

// --- Fake transport ----------------------------------------------------------

type fakeCITransport struct {
	calls  atomic.Int64
	bodies []string
}

func newFakeCITransport(bodies ...string) *fakeCITransport {
	return &fakeCITransport{bodies: bodies}
}

func (ft *fakeCITransport) assertCalls(t *testing.T, expected int64, msgAndArgs ...any) {
	t.Helper()
	assert.Equal(t, expected, ft.calls.Load(), msgAndArgs...)
}

func (ft *fakeCITransport) RoundTrip(r *http.Request) (*http.Response, error) {
	idx := int(ft.calls.Add(1)) - 1
	body := ft.bodies[min(idx, len(ft.bodies)-1)]

	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    r,
	}, nil
}

// --- Client constructor ------------------------------------------------------

// newKubemonConnectionInfoClient builds a real agclient.ClientImpl whose HTTP transport
// is the given fakeCITransport wrapped in a cache round-tripper with the specified TTL.
// PaasToken is set to t.Name() so each subtest has an isolated namespace in the global cache.
func newKubemonConnectionInfoClient(t *testing.T, transport http.RoundTripper, ttl time.Duration) agclient.Client {
	t.Helper()

	u, err := url.Parse("https://fake-dt.test")
	require.NoError(t, err)

	return agclient.NewClient(core.NewClient(core.Config{
		BaseURL:    u,
		HTTPClient: &http.Client{Transport: middleware.NewCacheRoundTripper(transport, ttl)},
		PaasToken:  t.Name(),
	}))
}

// --- Response body helpers ---------------------------------------------------

func kubemonConnectionInfoBody(tenantUUID, tenantToken, endpoints string) string {
	return fmt.Sprintf(
		`{"tenantUUID":%q,"tenantToken":%q,"communicationEndpoints":%q}`,
		tenantUUID, tenantToken, endpoints,
	)
}
