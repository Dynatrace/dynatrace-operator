package version

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

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

		synctest.Test(t, func(t *testing.T) {
			ctx := t.Context()
			dkStatus := &dk.Status
			versionClient := versionclientmock.NewClient(t)

			mockLatestAgentVersion(versionClient, latestAgentVersion, 3)
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

			// no change if probe not old enough
			previousProbe := *dkStatus.CodeModules.LastProbeTimestamp
			err = versionReconciler.ReconcileCodeModules(ctx, dk, nil, versionClient)
			require.NoError(t, err)
			assert.Equal(t, previousProbe, *dkStatus.CodeModules.LastProbeTimestamp)

			// change if probe old enough
			time.Sleep(15*time.Minute + 1*time.Second)

			err = versionReconciler.ReconcileCodeModules(ctx, dk, nil, versionClient)
			require.NoError(t, err)
			assert.NotEqual(t, previousProbe, *dkStatus.CodeModules.LastProbeTimestamp)
		})
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
	dk := dynakube.DynaKube{
		Spec: dynakube.DynaKubeSpec{
			OneAgent: oneagent.Spec{
				ClassicFullStack: &oneagent.HostInjectSpec{},
			},
		},
		Status: dynakube.DynaKubeStatus{
			OneAgent: oneagent.Status{
				VersionStatus: status.VersionStatus{
					Source: status.TenantRegistryVersionSource,
				},
			},
		},
	}

	t.Run("needs", func(t *testing.T) {
		dkCopy := dk.DeepCopy()
		reconciler := Reconciler{}
		assert.True(t, reconciler.needsUpdate(t.Context(), newOneAgentUpdater(dkCopy, fake.NewClient(), nil, nil), dkCopy))
	})
	t.Run("does not need", func(t *testing.T) {
		r := Reconciler{}
		assert.False(t, r.needsUpdate(t.Context(), newOneAgentUpdater(&dynakube.DynaKube{}, fake.NewClient(), nil, nil), &dynakube.DynaKube{}))
	})
	t.Run("does not need, because not old enough", func(t *testing.T) {
		oldImage := "repo.com:tag@sha256:123"
		newImage := "repo.com:tag"
		updatedDynakube := dk.DeepCopy()
		setOneAgentCustomImageStatus(updatedDynakube, oldImage)
		updatedDynakube.Spec.OneAgent.ClassicFullStack.Image = newImage
		now := metav1.Now()
		updatedDynakube.Status.OneAgent.LastProbeTimestamp = &now
		r := Reconciler{}
		assert.False(t, r.needsUpdate(t.Context(), newOneAgentUpdater(updatedDynakube, fake.NewClient(), nil, nil), updatedDynakube))
	})

	t.Run("needs, because source changed", func(t *testing.T) {
		updatedDynakube := dk.DeepCopy()
		setOneAgentCustomImageStatus(updatedDynakube, "")

		r := Reconciler{}
		assert.True(t, r.needsUpdate(t.Context(), newOneAgentUpdater(updatedDynakube, fake.NewClient(), nil, nil), updatedDynakube))
	})

	t.Run("needs, because custom image changed", func(t *testing.T) {
		oldImage := "repo.com:tag@sha256:123"
		newImage := "repo.com:newTag"
		updatedDynakube := dk.DeepCopy()
		updatedDynakube.Spec.OneAgent.ClassicFullStack.Image = newImage
		setOneAgentCustomImageStatus(updatedDynakube, oldImage)

		r := Reconciler{}
		assert.True(t, r.needsUpdate(t.Context(), newOneAgentUpdater(updatedDynakube, fake.NewClient(), nil, nil), updatedDynakube))
	})

	t.Run("needs, because custom version changed", func(t *testing.T) {
		oldVersion := "1.2.3.4-5"
		newVersion := "2.4.5.6-7"
		updatedDynakube := dk.DeepCopy()
		updatedDynakube.Spec.OneAgent.ClassicFullStack.Version = newVersion //nolint:staticcheck
		setOneAgentCustomVersionStatus(updatedDynakube, oldVersion)

		r := Reconciler{}
		assert.True(t, r.needsUpdate(t.Context(), newOneAgentUpdater(updatedDynakube, fake.NewClient(), nil, nil), updatedDynakube))
	})
}

func TestHasCustomFieldChanged(t *testing.T) {
	dk := dynakube.DynaKube{
		Spec: dynakube.DynaKubeSpec{
			OneAgent: oneagent.Spec{
				ClassicFullStack: &oneagent.HostInjectSpec{},
			},
		},
	}

	t.Run("version changed", func(t *testing.T) {
		oldVersion := "1.2.3.4-5"
		newVersion := "2.4.5.6-7"
		updatedDynakube := dk.DeepCopy()
		updatedDynakube.Spec.OneAgent.ClassicFullStack.Version = newVersion //nolint:staticcheck
		setOneAgentCustomVersionStatus(updatedDynakube, oldVersion)
		assert.True(t, hasCustomFieldChanged(t.Context(), newOneAgentUpdater(updatedDynakube, fake.NewClient(), nil, nil)))
	})

	t.Run("no change; version", func(t *testing.T) {
		version := "1.2.3.4-5"
		updatedDynakube := dk.DeepCopy()
		updatedDynakube.Spec.OneAgent.ClassicFullStack.Version = version //nolint:staticcheck
		setOneAgentCustomVersionStatus(updatedDynakube, version)
		assert.False(t, hasCustomFieldChanged(t.Context(), newOneAgentUpdater(updatedDynakube, fake.NewClient(), nil, nil)))
	})

	t.Run("image changed", func(t *testing.T) {
		oldImage := "repo.com:tag@sha256:123"
		newImage := "repo.com:Tag"
		updatedDynakube := dk.DeepCopy()
		updatedDynakube.Spec.OneAgent.ClassicFullStack.Image = newImage
		setOneAgentCustomImageStatus(updatedDynakube, oldImage)
		assert.True(t, hasCustomFieldChanged(t.Context(), newOneAgentUpdater(updatedDynakube, fake.NewClient(), nil, nil)))
	})

	t.Run("no change; image", func(t *testing.T) {
		oldImage := "repo.com:tag@sha256:123"
		newImage := "repo.com:tag"
		updatedDynakube := dk.DeepCopy()
		updatedDynakube.Spec.OneAgent.ClassicFullStack.Version = newImage //nolint:staticcheck
		setOneAgentCustomImageStatus(updatedDynakube, oldImage)
		assert.False(t, hasCustomFieldChanged(t.Context(), newOneAgentUpdater(updatedDynakube, fake.NewClient(), nil, nil)))
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

func setOneAgentCustomVersionStatus(dk *dynakube.DynaKube, version string) {
	dk.Status.OneAgent.Source = status.CustomVersionVersionSource
	dk.Status.OneAgent.Version = version
}

func setOneAgentCustomImageStatus(dk *dynakube.DynaKube, image string) {
	dk.Status.OneAgent.Source = status.CustomImageVersionSource
	dk.Status.OneAgent.ImageID = image
}

func mockLatestAgentVersion(mockClient *versionclientmock.Client, latestVersion string, expectedCalls int) {
	mockClient.EXPECT().GetLatestAgentVersion(anyCtx, mock.Anything, mock.Anything).Return(latestVersion, nil).Times(expectedCalls)
}

func mockLatestActiveGateVersion(mockClient *versionclientmock.Client, latestVersion string) {
	mockClient.EXPECT().GetLatestActiveGateVersion(anyCtx, mock.Anything).Return(latestVersion, nil).Once()
}
