package version

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	versionclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testNamespace      = "test-namespace"
	testDockerRegistry = "ENVIRONMENTID.live.dynatrace.com"
	testAPIURL         = "https://" + testDockerRegistry + "/api"

	latestActiveGateVersion = "1.2.3.4-56"
	latestOneAgentVersion   = "1.2.3.4-5"
)

var anyCtx = mock.MatchedBy(func(context.Context) bool { return true })

func TestReconcile(t *testing.T) {
	ctx := t.Context()
	latestAgentVersion := "1.2.3.4-5"
	dynakubeTemplate := dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace},
		Spec: dynakube.DynaKubeSpec{
			APIURL: testAPIURL,
			OneAgent: oneagent.Spec{
				CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
			},
			ActiveGate: activegate.Spec{
				Capabilities: []activegate.CapabilityDisplayName{
					activegate.KubeMonCapability.DisplayName,
				},
			},
		},
	}

	t.Run("no update if hash provider returns error", func(t *testing.T) {
		versionClient := versionclientmock.NewClient(t)
		versionClient.EXPECT().GetLatestActiveGateVersion(anyCtx, mock.Anything).Return("", errors.New("Something wrong happened"))

		versionReconciler := Reconciler{
			apiReader: fake.NewClient(),
		}
		dk := dynakubeTemplate.DeepCopy()
		err := versionReconciler.ReconcileActiveGate(ctx, dk, nil, versionClient)
		require.Error(t, err)

		condition := meta.FindStatusCondition(dk.Status.Conditions, activeGateVersionConditionType)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
		assert.Equal(t, k8sconditions.DynatraceAPIErrorReason, condition.Reason)
	})

	t.Run("all image versions were updated", func(t *testing.T) {
		dk := dynakubeTemplate.DeepCopy()
		fakeClient := fake.NewClient()

		setupPullSecret(t, fakeClient, *dk)

		ctx := t.Context()
		dkStatus := &dk.Status
		versionClient := versionclientmock.NewClient(t)

		mockLatestAgentVersion(versionClient, latestAgentVersion, 2)
		mockLatestActiveGateVersion(versionClient, latestActiveGateVersion)

		versionReconciler := Reconciler{
			apiReader: fakeClient,
		}
		err := versionReconciler.ReconcileCodeModules(ctx, dk, nil, versionClient)
		require.NoError(t, err)
		err = versionReconciler.ReconcileActiveGate(ctx, dk, nil, versionClient)
		require.NoError(t, err)
		err = versionReconciler.ReconcileOneAgent(ctx, dk, nil, versionClient)
		require.NoError(t, err)

		condition := meta.FindStatusCondition(dk.Status.Conditions, activeGateVersionConditionType)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, verifiedReason, condition.Reason)
		assert.Equal(t, "Version verified for component.", condition.Message)

		assertStatusBasedOnTenantRegistry(t, dk.ActiveGate().GetDefaultImage(latestActiveGateVersion), latestActiveGateVersion, dkStatus.ActiveGate.VersionStatus)
		assertStatusBasedOnTenantRegistry(t, dk.OneAgent().GetDefaultImage(latestOneAgentVersion), latestOneAgentVersion, dkStatus.OneAgent.VersionStatus)
		assert.Equal(t, latestAgentVersion, dkStatus.CodeModules.Version)
	})
}

func TestUpdateVersionStatuses(t *testing.T) {
	ctx := t.Context()

	t.Run("empty version info + failing reconcile => return error", func(t *testing.T) {
		versionReconciler := Reconciler{
			apiReader: fake.NewClient(),
		}
		updater := newFailingUpdater(t)
		updater.EXPECT().Target().Return(&status.VersionStatus{}).Times(2)
		err := versionReconciler.updateVersionStatuses(ctx, updater, &dynakube.DynaKube{})
		require.Error(t, err)
	})

	t.Run("version info (.Version) set + failing reconcile => return nil", func(t *testing.T) {
		versionReconciler := Reconciler{
			apiReader: fake.NewClient(),
		}
		updater := newFailingUpdater(t)
		updater.EXPECT().Target().Return(&status.VersionStatus{Version: "1.2.3"}).Times(2)
		err := versionReconciler.updateVersionStatuses(ctx, updater, &dynakube.DynaKube{})
		require.NoError(t, err)
	})

	t.Run("version info (.ImageID) set + failing reconcile => return nil", func(t *testing.T) {
		versionReconciler := Reconciler{
			apiReader: fake.NewClient(),
		}
		updater := newFailingUpdater(t)
		updater.EXPECT().Target().Return(&status.VersionStatus{ImageID: "some-image"}).Once()
		err := versionReconciler.updateVersionStatuses(ctx, updater, &dynakube.DynaKube{})
		require.NoError(t, err)
	})
}

func TestNeedsUpdate(t *testing.T) {
	t.Run("needs", func(t *testing.T) {
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					ClassicFullStack: &oneagent.HostInjectSpec{},
				},
			},
		}
		reconciler := Reconciler{}
		assert.True(t, reconciler.needsUpdate(t.Context(), newOneAgentUpdater(&dk, fake.NewClient(), nil, nil)))
	})
	t.Run("does not need", func(t *testing.T) {
		r := Reconciler{}
		assert.False(t, r.needsUpdate(t.Context(), newOneAgentUpdater(&dynakube.DynaKube{}, fake.NewClient(), nil, nil)))
	})
}

func setupPullSecret(t *testing.T, fakeClient client.Client, dk dynakube.DynaKube) {
	t.Helper()
	err := createTestPullSecret(t, fakeClient, dk)
	require.NoError(t, err)
}

func createTestPullSecret(t *testing.T, fakeClient client.Client, dk dynakube.DynaKube) error {
	return fakeClient.Create(t.Context(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: dk.Namespace,
			Name:      dk.TenantRegistryPullSecretName(),
		},
		Data: map[string][]byte{
			".dockerconfigjson": []byte("{}"),
		},
	})
}

func mockLatestAgentVersion(mockClient *versionclientmock.Client, latestVersion string, expectedCalls int) {
	mockClient.EXPECT().GetLatestAgentVersion(anyCtx, mock.Anything, mock.Anything).Return(latestVersion, nil).Times(expectedCalls)
}

func mockLatestActiveGateVersion(mockClient *versionclientmock.Client, latestVersion string) {
	mockClient.EXPECT().GetLatestActiveGateVersion(anyCtx, mock.Anything).Return(latestVersion, nil).Once()
}
