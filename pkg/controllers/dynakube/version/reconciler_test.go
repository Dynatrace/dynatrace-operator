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
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testNamespace      = "test-namespace"
	testDockerRegistry = "ENVIRONMENTID.live.dynatrace.com"
	testAPIURL         = "https://" + testDockerRegistry + "/api"
)

var anyCtx = mock.MatchedBy(func(context.Context) bool { return true })

func TestReconcile(t *testing.T) {
	ctx := t.Context()
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
		mockClient := dtclientmock.NewClient(t)
		mockClient.EXPECT().GetLatestActiveGateVersion(anyCtx, mock.Anything).Return("", errors.New("Something wrong happened"))

		versionReconciler := reconciler{
			dtClient:     mockClient,
			apiReader:    fake.NewClient(),
			timeProvider: timeprovider.New().Freeze(),
		}
		dk := dynakubeTemplate.DeepCopy()
		err := versionReconciler.ReconcileActiveGate(ctx, dk)
		require.Error(t, err)

		condition := meta.FindStatusCondition(dk.Status.Conditions, activeGateVersionConditionType)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
		assert.Equal(t, k8sconditions.DynatraceAPIErrorReason, condition.Reason)
	})
}

func TestUpdateVersionStatuses(t *testing.T) {
	ctx := t.Context()

	t.Run("empty version info + failing reconcile => return error", func(t *testing.T) {
		mockClient := dtclientmock.NewClient(t)
		versionReconciler := reconciler{
			dtClient:     mockClient,
			apiReader:    fake.NewClient(),
			timeProvider: timeprovider.New().Freeze(),
		}
		err := versionReconciler.updateVersionStatuses(ctx, newFailingUpdater(t, &status.VersionStatus{}), &dynakube.DynaKube{})
		require.Error(t, err)
	})

	t.Run("version info (.Version) set + failing reconcile => return nil", func(t *testing.T) {
		mockClient := dtclientmock.NewClient(t)
		versionReconciler := reconciler{
			dtClient:     mockClient,
			apiReader:    fake.NewClient(),
			timeProvider: timeprovider.New().Freeze(),
		}
		err := versionReconciler.updateVersionStatuses(ctx, newFailingUpdater(t, &status.VersionStatus{Version: "1.2.3"}), &dynakube.DynaKube{})
		require.NoError(t, err)
	})

	t.Run("version info (.ImageID) set + failing reconcile => return nil", func(t *testing.T) {
		mockClient := dtclientmock.NewClient(t)
		versionReconciler := reconciler{
			dtClient:     mockClient,
			apiReader:    fake.NewClient(),
			timeProvider: timeprovider.New().Freeze(),
		}
		err := versionReconciler.updateVersionStatuses(ctx, newFailingUpdater(t, &status.VersionStatus{ImageID: "1.2.3"}), &dynakube.DynaKube{})
		require.NoError(t, err)
	})
}

func TestNeedsUpdate(t *testing.T) {
	timeProvider := timeprovider.New().Freeze()

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
		reconciler := reconciler{
			timeProvider: timeProvider,
		}
		assert.True(t, reconciler.needsUpdate(newOneAgentUpdater(dkCopy, fake.NewClient(), nil), dkCopy))
	})
	t.Run("does not need", func(t *testing.T) {
		r := reconciler{
			timeProvider: timeProvider,
		}
		assert.False(t, r.needsUpdate(newOneAgentUpdater(&dynakube.DynaKube{}, fake.NewClient(), nil), &dynakube.DynaKube{}))
	})
	t.Run("does not need, because not old enough", func(t *testing.T) {
		oldImage := "repo.com:tag@sha256:123"
		newImage := "repo.com:tag"
		updatedDynakube := dk.DeepCopy()
		setOneAgentCustomImageStatus(updatedDynakube, oldImage)
		updatedDynakube.Spec.OneAgent.ClassicFullStack.Image = newImage
		updatedDynakube.Status.OneAgent.LastProbeTimestamp = timeProvider.Now()
		r := reconciler{
			timeProvider: timeProvider,
		}
		assert.False(t, r.needsUpdate(newOneAgentUpdater(updatedDynakube, fake.NewClient(), nil), updatedDynakube))
	})

	t.Run("needs, because source changed", func(t *testing.T) {
		updatedDynakube := dk.DeepCopy()
		setOneAgentCustomImageStatus(updatedDynakube, "")

		r := reconciler{
			timeProvider: timeProvider,
		}
		assert.True(t, r.needsUpdate(newOneAgentUpdater(updatedDynakube, fake.NewClient(), nil), updatedDynakube))
	})

	t.Run("needs, because custom image changed", func(t *testing.T) {
		oldImage := "repo.com:tag@sha256:123"
		newImage := "repo.com:newTag"
		updatedDynakube := dk.DeepCopy()
		updatedDynakube.Spec.OneAgent.ClassicFullStack.Image = newImage
		setOneAgentCustomImageStatus(updatedDynakube, oldImage)

		r := reconciler{
			timeProvider: timeProvider,
		}
		assert.True(t, r.needsUpdate(newOneAgentUpdater(updatedDynakube, fake.NewClient(), nil), updatedDynakube))
	})

	t.Run("needs, because custom version changed", func(t *testing.T) {
		oldVersion := "1.2.3.4-5"
		newVersion := "2.4.5.6-7"
		updatedDynakube := dk.DeepCopy()
		updatedDynakube.Spec.OneAgent.ClassicFullStack.Version = newVersion //nolint:staticcheck
		setOneAgentCustomVersionStatus(updatedDynakube, oldVersion)

		r := reconciler{
			timeProvider: timeProvider,
		}
		assert.True(t, r.needsUpdate(newOneAgentUpdater(updatedDynakube, fake.NewClient(), nil), updatedDynakube))
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
		assert.True(t, hasCustomFieldChanged(newOneAgentUpdater(updatedDynakube, fake.NewClient(), nil)))
	})

	t.Run("no change; version", func(t *testing.T) {
		version := "1.2.3.4-5"
		updatedDynakube := dk.DeepCopy()
		updatedDynakube.Spec.OneAgent.ClassicFullStack.Version = version //nolint:staticcheck
		setOneAgentCustomVersionStatus(updatedDynakube, version)
		assert.False(t, hasCustomFieldChanged(newOneAgentUpdater(updatedDynakube, fake.NewClient(), nil)))
	})

	t.Run("image changed", func(t *testing.T) {
		oldImage := "repo.com:tag@sha256:123"
		newImage := "repo.com:Tag"
		updatedDynakube := dk.DeepCopy()
		updatedDynakube.Spec.OneAgent.ClassicFullStack.Image = newImage
		setOneAgentCustomImageStatus(updatedDynakube, oldImage)
		assert.True(t, hasCustomFieldChanged(newOneAgentUpdater(updatedDynakube, fake.NewClient(), nil)))
	})

	t.Run("no change; image", func(t *testing.T) {
		oldImage := "repo.com:tag@sha256:123"
		newImage := "repo.com:tag"
		updatedDynakube := dk.DeepCopy()
		updatedDynakube.Spec.OneAgent.ClassicFullStack.Version = newImage //nolint:staticcheck
		setOneAgentCustomImageStatus(updatedDynakube, oldImage)
		assert.False(t, hasCustomFieldChanged(newOneAgentUpdater(updatedDynakube, fake.NewClient(), nil)))
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

func mockLatestAgentVersion(mockClient *dtclientmock.Client, latestVersion string, expectedCalls int) {
	mockClient.EXPECT().GetLatestAgentVersion(anyCtx, mock.Anything, mock.Anything).Return(latestVersion, nil).Times(expectedCalls)
}
