package version

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
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
	testApiUrl         = "https://" + testDockerRegistry + "/api"

	latestActiveGateVersion = "1.2.3.4-56"
)

func TestReconcile(t *testing.T) {
	ctx := context.Background()
	latestAgentVersion := "1.2.3.4-5"
	dynakubeTemplate := dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace},
		Spec: dynakube.DynaKubeSpec{
			APIURL: testApiUrl,
			OneAgent: dynakube.OneAgentSpec{
				CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{},
			},
			ActiveGate: activegate.Spec{
				Capabilities: []activegate.CapabilityDisplayName{
					activegate.CapabilityDisplayName(activegate.KubeMonCapability.ShortName),
				},
			},
		},
	}

	t.Run("no update if hash provider returns error", func(t *testing.T) {
		mockClient := dtclientmock.NewClient(t)
		mockClient.On("GetLatestActiveGateVersion", mock.AnythingOfType("context.backgroundCtx"), mock.Anything).Return("", errors.New("Something wrong happened"))

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
		assert.Equal(t, conditions.DynatraceApiErrorReason, condition.Reason)
	})

	t.Run("all image versions were updated", func(t *testing.T) {
		testActiveGateImage := getTestActiveGateImageInfo()
		testOneAgentImage := getTestOneAgentImageInfo()
		dk := dynakubeTemplate.DeepCopy()
		fakeClient := fake.NewClient()
		timeProvider := timeprovider.New().Freeze()

		setupPullSecret(t, fakeClient, *dk)

		dkStatus := &dk.Status
		mockClient := dtclientmock.NewClient(t)
		mockLatestAgentVersion(mockClient, latestAgentVersion)
		mockLatestActiveGateVersion(mockClient, latestActiveGateVersion)

		versionReconciler := reconciler{
			apiReader:    fakeClient,
			timeProvider: timeProvider,
			dtClient:     mockClient,
		}
		err := versionReconciler.ReconcileCodeModules(ctx, dk)
		require.NoError(t, err)
		err = versionReconciler.ReconcileActiveGate(ctx, dk)
		require.NoError(t, err)
		err = versionReconciler.ReconcileOneAgent(ctx, dk)
		require.NoError(t, err)

		condition := meta.FindStatusCondition(dk.Status.Conditions, activeGateVersionConditionType)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, verifiedReason, condition.Reason)
		assert.Equal(t, "Version verified for component.", condition.Message)

		assertStatusBasedOnTenantRegistry(t, dk.ActiveGate().GetDefaultImage(testActiveGateImage.Tag), testActiveGateImage.Tag, dkStatus.ActiveGate.VersionStatus)
		assertStatusBasedOnTenantRegistry(t, dk.DefaultOneAgentImage(testOneAgentImage.Tag), testOneAgentImage.Tag, dkStatus.OneAgent.VersionStatus)
		assert.Equal(t, latestAgentVersion, dkStatus.CodeModules.VersionStatus.Version)

		// no change if probe not old enough
		previousProbe := *dkStatus.CodeModules.VersionStatus.LastProbeTimestamp
		err = versionReconciler.ReconcileCodeModules(ctx, dk)
		require.NoError(t, err)
		assert.Equal(t, previousProbe, *dkStatus.CodeModules.VersionStatus.LastProbeTimestamp)

		// change if probe old enough
		changeTime(timeProvider, 15*time.Minute+1*time.Second)

		err = versionReconciler.ReconcileCodeModules(ctx, dk)
		require.NoError(t, err)
		assert.NotEqual(t, previousProbe, *dkStatus.CodeModules.VersionStatus.LastProbeTimestamp)
	})

	t.Run("public-registry", func(t *testing.T) {
		testActiveGateImage := getTestActiveGateImageInfo()
		testOneAgentImage := getTestOneAgentImageInfo()
		testCodeModulesImage := getTestCodeModulesImage()
		dk := dynakubeTemplate.DeepCopy()
		enablePublicRegistry(dk)

		fakeClient := fake.NewClient()
		setupPullSecret(t, fakeClient, *dk)

		dkStatus := &dk.Status

		mockClient := dtclientmock.NewClient(t)
		mockActiveGateImageInfo(mockClient, testActiveGateImage)
		mockCodeModulesImageInfo(mockClient, testCodeModulesImage)
		mockOneAgentImageInfo(mockClient, testOneAgentImage)

		versionReconciler := reconciler{
			apiReader:    fakeClient,
			timeProvider: timeprovider.New().Freeze(),
			dtClient:     mockClient,
		}
		err := versionReconciler.ReconcileCodeModules(ctx, dk)
		require.NoError(t, err)
		err = versionReconciler.ReconcileActiveGate(ctx, dk)
		require.NoError(t, err)
		err = versionReconciler.ReconcileOneAgent(ctx, dk)
		require.NoError(t, err)
		require.NoError(t, err)

		condition := meta.FindStatusCondition(dk.Status.Conditions, activeGateVersionConditionType)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, verifiedReason, condition.Reason)
		assert.Equal(t, "Version verified for component.", condition.Message)

		assert.Equal(t, testActiveGateImage.String(), dkStatus.ActiveGate.VersionStatus.ImageID)
		assert.Equal(t, testActiveGateImage.Tag, dkStatus.ActiveGate.VersionStatus.Version)
		assert.Equal(t, testOneAgentImage.String(), dkStatus.OneAgent.VersionStatus.ImageID)
		assert.Equal(t, testOneAgentImage.Tag, dkStatus.OneAgent.VersionStatus.Version)
		assert.Equal(t, testCodeModulesImage.String(), dkStatus.CodeModules.VersionStatus.ImageID)
		assert.Equal(t, testCodeModulesImage.Tag, dkStatus.CodeModules.VersionStatus.Version)
	})
}

func TestUpdateVersionStatuses(t *testing.T) {
	ctx := context.Background()

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
			OneAgent: dynakube.OneAgentSpec{
				ClassicFullStack: &dynakube.HostInjectSpec{},
			},
			DynatraceApiRequestThreshold: dynakube.DefaultMinRequestThresholdMinutes,
		},
		Status: dynakube.DynaKubeStatus{
			OneAgent: dynakube.OneAgentStatus{
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
		updatedDynakube.Spec.OneAgent.ClassicFullStack.Version = newVersion
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
			OneAgent: dynakube.OneAgentSpec{
				ClassicFullStack: &dynakube.HostInjectSpec{},
			},
		},
	}

	t.Run("version changed", func(t *testing.T) {
		oldVersion := "1.2.3.4-5"
		newVersion := "2.4.5.6-7"
		updatedDynakube := dk.DeepCopy()
		updatedDynakube.Spec.OneAgent.ClassicFullStack.Version = newVersion
		setOneAgentCustomVersionStatus(updatedDynakube, oldVersion)
		assert.True(t, hasCustomFieldChanged(newOneAgentUpdater(updatedDynakube, fake.NewClient(), nil)))
	})

	t.Run("no change; version", func(t *testing.T) {
		version := "1.2.3.4-5"
		updatedDynakube := dk.DeepCopy()
		updatedDynakube.Spec.OneAgent.ClassicFullStack.Version = version
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
		updatedDynakube.Spec.OneAgent.ClassicFullStack.Version = newImage
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

func getTestOneAgentImageInfo() dtclient.LatestImageInfo {
	return dtclient.LatestImageInfo{
		Source: testDockerRegistry + "/linux/oneagent",
		Tag:    "1.2.3.4-5",
	}
}

func getTestActiveGateImageInfo() dtclient.LatestImageInfo {
	return dtclient.LatestImageInfo{
		Source: testDockerRegistry + "/linux/activegate",
		Tag:    latestActiveGateVersion,
	}
}

func getTestCodeModulesImage() dtclient.LatestImageInfo {
	return dtclient.LatestImageInfo{
		Source: testDockerRegistry + "/linux/codemodules",
		Tag:    "1.2.3.4-5",
	}
}

func setupPullSecret(t *testing.T, fakeClient client.Client, dk dynakube.DynaKube) {
	err := createTestPullSecret(fakeClient, dk)
	require.NoError(t, err)
}

func changeTime(timeProvider *timeprovider.Provider, duration time.Duration) {
	timeProvider.Set(timeProvider.Now().Add(duration))
}

func createTestPullSecret(fakeClient client.Client, dk dynakube.DynaKube) error {
	return fakeClient.Create(context.TODO(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: dk.Namespace,
			Name:      dk.Name + dtpullsecret.PullSecretSuffix,
		},
		Data: map[string][]byte{
			".dockerconfigjson": []byte("{}"),
		},
	})
}

func mockActiveGateImageInfo(mockClient *dtclientmock.Client, imageInfo dtclient.LatestImageInfo) {
	mockClient.On("GetLatestActiveGateImage", mock.AnythingOfType("context.backgroundCtx")).Return(&imageInfo, nil)
}

func mockCodeModulesImageInfo(mockClient *dtclientmock.Client, imageInfo dtclient.LatestImageInfo) {
	mockClient.On("GetLatestCodeModulesImage", mock.AnythingOfType("context.backgroundCtx")).Return(&imageInfo, nil)
}

func mockOneAgentImageInfo(mockClient *dtclientmock.Client, imageInfo dtclient.LatestImageInfo) {
	mockClient.On("GetLatestOneAgentImage", mock.AnythingOfType("context.backgroundCtx")).Return(&imageInfo, nil)
}

func mockLatestAgentVersion(mockClient *dtclientmock.Client, latestVersion string) {
	mockClient.On("GetLatestAgentVersion", mock.AnythingOfType("context.backgroundCtx"), mock.Anything, mock.Anything).Return(latestVersion, nil)
}

func mockLatestActiveGateVersion(mockClient *dtclientmock.Client, latestVersion string) {
	mockClient.On("GetLatestActiveGateVersion", mock.AnythingOfType("context.backgroundCtx"), mock.Anything).Return(latestVersion, nil)
}

func createErrorDTClient(t *testing.T) dtclient.Client {
	mockClient := dtclientmock.NewClient(t)
	mockClient.On("GetLatestAgentVersion", mock.AnythingOfType("context.backgroundCtx"), mock.Anything, mock.Anything).Return("", errors.New("BOOM")).Maybe()
	mockClient.On("GetLatestActiveGateVersion", mock.AnythingOfType("context.backgroundCtx"), mock.Anything).Return("", errors.New("BOOM")).Maybe()
	mockClient.On("GetLatestOneAgentImage", mock.AnythingOfType("context.backgroundCtx")).Return(nil, errors.New("BOOM")).Maybe()
	mockClient.On("GetLatestCodeModulesImage", mock.AnythingOfType("context.backgroundCtx")).Return(nil, errors.New("BOOM")).Maybe()
	mockClient.On("GetLatestActiveGateImage", mock.AnythingOfType("context.backgroundCtx")).Return(nil, errors.New("BOOM")).Maybe()

	return mockClient
}
