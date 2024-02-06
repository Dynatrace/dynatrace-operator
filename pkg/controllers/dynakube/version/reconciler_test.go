package version

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	mockedclient "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
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
	dynakubeTemplate := dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: testApiUrl,
			OneAgent: dynatracev1beta1.OneAgentSpec{
				CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{},
			},
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				Capabilities: []dynatracev1beta1.CapabilityDisplayName{
					dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.KubeMonCapability.ShortName),
				},
			},
		},
	}

	t.Run("no update if hash provider returns error", func(t *testing.T) {
		mockClient := mockedclient.NewClient(t)
		mockClient.On("GetLatestActiveGateVersion", mock.Anything).Return("", errors.New("Something wrong happened"))

		versionReconciler := reconciler{
			dtClient:     mockClient,
			apiReader:    fake.NewClient(),
			fs:           afero.Afero{Fs: afero.NewMemMapFs()},
			timeProvider: timeprovider.New().Freeze(),
		}
		err := versionReconciler.ReconcileActiveGate(ctx, dynakubeTemplate.DeepCopy())
		assert.Error(t, err)
	})

	t.Run("all image versions were updated", func(t *testing.T) {
		testActiveGateImage := getTestActiveGateImageInfo()
		testOneAgentImage := getTestOneAgentImageInfo()
		dynakube := dynakubeTemplate.DeepCopy()
		fakeClient := fake.NewClient()
		timeProvider := timeprovider.New().Freeze()

		setupPullSecret(t, fakeClient, *dynakube)

		dkStatus := &dynakube.Status
		mockClient := mockedclient.NewClient(t)
		mockLatestAgentVersion(mockClient, latestAgentVersion)
		mockLatestActiveGateVersion(mockClient, latestActiveGateVersion)

		versionReconciler := reconciler{
			apiReader:    fakeClient,
			fs:           afero.Afero{Fs: afero.NewMemMapFs()},
			timeProvider: timeProvider,
			dtClient:     mockClient,
		}
		err := versionReconciler.ReconcileCodeModules(ctx, dynakube)
		require.NoError(t, err)
		err = versionReconciler.ReconcileActiveGate(ctx, dynakube)
		require.NoError(t, err)
		err = versionReconciler.ReconcileOneAgent(ctx, dynakube)
		require.NoError(t, err)

		assertStatusBasedOnTenantRegistry(t, dynakube.DefaultActiveGateImage(), testActiveGateImage.Tag, dkStatus.ActiveGate.VersionStatus)
		assertStatusBasedOnTenantRegistry(t, dynakube.DefaultOneAgentImage(), testOneAgentImage.Tag, dkStatus.OneAgent.VersionStatus)
		assert.Equal(t, latestAgentVersion, dkStatus.CodeModules.VersionStatus.Version)

		// no change if probe not old enough
		previousProbe := *dkStatus.CodeModules.VersionStatus.LastProbeTimestamp
		err = versionReconciler.ReconcileCodeModules(ctx, dynakube)
		require.NoError(t, err)
		assert.Equal(t, previousProbe, *dkStatus.CodeModules.VersionStatus.LastProbeTimestamp)

		// change if probe old enough
		changeTime(timeProvider, 15*time.Minute+1*time.Second)

		err = versionReconciler.ReconcileCodeModules(ctx, dynakube)
		require.NoError(t, err)
		assert.NotEqual(t, previousProbe, *dkStatus.CodeModules.VersionStatus.LastProbeTimestamp)
	})

	t.Run("public-registry", func(t *testing.T) {
		testActiveGateImage := getTestActiveGateImageInfo()
		testOneAgentImage := getTestOneAgentImageInfo()
		testCodeModulesImage := getTestCodeModulesImage()
		dynakube := dynakubeTemplate.DeepCopy()
		enablePublicRegistry(dynakube)

		fakeClient := fake.NewClient()
		setupPullSecret(t, fakeClient, *dynakube)

		dkStatus := &dynakube.Status

		mockClient := mockedclient.NewClient(t)
		mockActiveGateImageInfo(mockClient, testActiveGateImage)
		mockCodeModulesImageInfo(mockClient, testCodeModulesImage)
		mockOneAgentImageInfo(mockClient, testOneAgentImage)

		versionReconciler := reconciler{
			apiReader:    fakeClient,
			fs:           afero.Afero{Fs: afero.NewMemMapFs()},
			timeProvider: timeprovider.New().Freeze(),
			dtClient:     mockClient,
		}
		err := versionReconciler.ReconcileCodeModules(ctx, dynakube)
		require.NoError(t, err)
		err = versionReconciler.ReconcileActiveGate(ctx, dynakube)
		require.NoError(t, err)
		err = versionReconciler.ReconcileOneAgent(ctx, dynakube)
		require.NoError(t, err)
		require.NoError(t, err)

		assert.Equal(t, testActiveGateImage.String(), dkStatus.ActiveGate.VersionStatus.ImageID)
		assert.Equal(t, testActiveGateImage.Tag, dkStatus.ActiveGate.VersionStatus.Version)
		assert.Equal(t, testOneAgentImage.String(), dkStatus.OneAgent.VersionStatus.ImageID)
		assert.Equal(t, testOneAgentImage.Tag, dkStatus.OneAgent.VersionStatus.Version)
		assert.Equal(t, testCodeModulesImage.String(), dkStatus.CodeModules.VersionStatus.ImageID)
		assert.Equal(t, testCodeModulesImage.Tag, dkStatus.CodeModules.VersionStatus.Version)
	})
}

func TestNeedsUpdate(t *testing.T) {
	timeProvider := timeprovider.New().Freeze()

	dynakube := dynatracev1beta1.DynaKube{
		Spec: dynatracev1beta1.DynaKubeSpec{
			OneAgent: dynatracev1beta1.OneAgentSpec{
				ClassicFullStack: &dynatracev1beta1.HostInjectSpec{},
			},
		},
		Status: dynatracev1beta1.DynaKubeStatus{
			OneAgent: dynatracev1beta1.OneAgentStatus{
				VersionStatus: status.VersionStatus{
					Source: status.TenantRegistryVersionSource,
				},
			},
		},
	}

	t.Run("needs", func(t *testing.T) {
		updatedDynakube := dynakube.DeepCopy()
		reconciler := reconciler{
			timeProvider: timeProvider,
		}
		assert.True(t, reconciler.needsUpdate(newOneAgentUpdater(updatedDynakube, fake.NewClient(), nil), updatedDynakube))
	})
	t.Run("does not need", func(t *testing.T) {
		reconciler := reconciler{
			timeProvider: timeProvider,
		}
		assert.False(t, reconciler.needsUpdate(newOneAgentUpdater(&dynatracev1beta1.DynaKube{}, fake.NewClient(), nil), &dynatracev1beta1.DynaKube{}))
	})
	t.Run("does not need, because not old enough", func(t *testing.T) {
		oldImage := "repo.com:tag@sha256:123"
		newImage := "repo.com:tag"
		updatedDynakube := dynakube.DeepCopy()
		setOneAgentCustomImageStatus(updatedDynakube, oldImage)
		updatedDynakube.Spec.OneAgent.ClassicFullStack.Image = newImage
		updatedDynakube.Status.OneAgent.LastProbeTimestamp = timeProvider.Now()
		reconciler := reconciler{
			timeProvider: timeProvider,
		}
		assert.False(t, reconciler.needsUpdate(newOneAgentUpdater(updatedDynakube, fake.NewClient(), nil), updatedDynakube))
	})

	t.Run("needs, because source changed", func(t *testing.T) {
		updatedDynakube := dynakube.DeepCopy()
		setOneAgentCustomImageStatus(updatedDynakube, "")

		reconciler := reconciler{
			timeProvider: timeProvider,
		}
		assert.True(t, reconciler.needsUpdate(newOneAgentUpdater(updatedDynakube, fake.NewClient(), nil), updatedDynakube))
	})

	t.Run("needs, because custom image changed", func(t *testing.T) {
		oldImage := "repo.com:tag@sha256:123"
		newImage := "repo.com:newTag"
		updatedDynakube := dynakube.DeepCopy()
		updatedDynakube.Spec.OneAgent.ClassicFullStack.Image = newImage
		setOneAgentCustomImageStatus(updatedDynakube, oldImage)

		reconciler := reconciler{
			timeProvider: timeProvider,
		}
		assert.True(t, reconciler.needsUpdate(newOneAgentUpdater(updatedDynakube, fake.NewClient(), nil), updatedDynakube))
	})

	t.Run("needs, because custom version changed", func(t *testing.T) {
		oldVersion := "1.2.3.4-5"
		newVersion := "2.4.5.6-7"
		updatedDynakube := dynakube.DeepCopy()
		updatedDynakube.Spec.OneAgent.ClassicFullStack.Version = newVersion
		setOneAgentCustomVersionStatus(updatedDynakube, oldVersion)

		reconciler := reconciler{
			timeProvider: timeProvider,
		}
		assert.True(t, reconciler.needsUpdate(newOneAgentUpdater(updatedDynakube, fake.NewClient(), nil), updatedDynakube))
	})
}

func TestHasCustomFieldChanged(t *testing.T) {
	dynakube := dynatracev1beta1.DynaKube{
		Spec: dynatracev1beta1.DynaKubeSpec{
			OneAgent: dynatracev1beta1.OneAgentSpec{
				ClassicFullStack: &dynatracev1beta1.HostInjectSpec{},
			},
		},
	}

	t.Run("version changed", func(t *testing.T) {
		oldVersion := "1.2.3.4-5"
		newVersion := "2.4.5.6-7"
		updatedDynakube := dynakube.DeepCopy()
		updatedDynakube.Spec.OneAgent.ClassicFullStack.Version = newVersion
		setOneAgentCustomVersionStatus(updatedDynakube, oldVersion)
		assert.True(t, hasCustomFieldChanged(newOneAgentUpdater(updatedDynakube, fake.NewClient(), nil)))
	})

	t.Run("no change; version", func(t *testing.T) {
		version := "1.2.3.4-5"
		updatedDynakube := dynakube.DeepCopy()
		updatedDynakube.Spec.OneAgent.ClassicFullStack.Version = version
		setOneAgentCustomVersionStatus(updatedDynakube, version)
		assert.False(t, hasCustomFieldChanged(newOneAgentUpdater(updatedDynakube, fake.NewClient(), nil)))
	})

	t.Run("image changed", func(t *testing.T) {
		oldImage := "repo.com:tag@sha256:123"
		newImage := "repo.com:Tag"
		updatedDynakube := dynakube.DeepCopy()
		updatedDynakube.Spec.OneAgent.ClassicFullStack.Image = newImage
		setOneAgentCustomImageStatus(updatedDynakube, oldImage)
		assert.True(t, hasCustomFieldChanged(newOneAgentUpdater(updatedDynakube, fake.NewClient(), nil)))
	})

	t.Run("no change; image", func(t *testing.T) {
		oldImage := "repo.com:tag@sha256:123"
		newImage := "repo.com:tag"
		updatedDynakube := dynakube.DeepCopy()
		updatedDynakube.Spec.OneAgent.ClassicFullStack.Version = newImage
		setOneAgentCustomImageStatus(updatedDynakube, oldImage)
		assert.False(t, hasCustomFieldChanged(newOneAgentUpdater(updatedDynakube, fake.NewClient(), nil)))
	})
}

func setOneAgentCustomVersionStatus(dynakube *dynatracev1beta1.DynaKube, version string) {
	dynakube.Status.OneAgent.Source = status.CustomVersionVersionSource
	dynakube.Status.OneAgent.Version = version
}

func setOneAgentCustomImageStatus(dynakube *dynatracev1beta1.DynaKube, image string) {
	dynakube.Status.OneAgent.Source = status.CustomImageVersionSource
	dynakube.Status.OneAgent.ImageID = image
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

func setupPullSecret(t *testing.T, fakeClient client.Client, dynakube dynatracev1beta1.DynaKube) {
	err := createTestPullSecret(fakeClient, dynakube)
	require.NoError(t, err)
}

func changeTime(timeProvider *timeprovider.Provider, duration time.Duration) {
	newTime := metav1.NewTime(timeProvider.Now().Add(duration))
	timeProvider.Set(&newTime)
}

func createTestPullSecret(fakeClient client.Client, dynakube dynatracev1beta1.DynaKube) error {
	return fakeClient.Create(context.TODO(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: dynakube.Namespace,
			Name:      dynakube.Name + dtpullsecret.PullSecretSuffix,
		},
		Data: map[string][]byte{
			".dockerconfigjson": []byte("{}"),
		},
	})
}

func mockActiveGateImageInfo(mockClient *mockedclient.Client, imageInfo dtclient.LatestImageInfo) {
	mockClient.On("GetLatestActiveGateImage").Return(&imageInfo, nil)
}

func mockCodeModulesImageInfo(mockClient *mockedclient.Client, imageInfo dtclient.LatestImageInfo) {
	mockClient.On("GetLatestCodeModulesImage").Return(&imageInfo, nil)
}

func mockOneAgentImageInfo(mockClient *mockedclient.Client, imageInfo dtclient.LatestImageInfo) {
	mockClient.On("GetLatestOneAgentImage").Return(&imageInfo, nil)
}

func mockLatestAgentVersion(mockClient *mockedclient.Client, latestVersion string) {
	mockClient.On("GetLatestAgentVersion", mock.Anything, mock.Anything).Return(latestVersion, nil)
}

func mockLatestActiveGateVersion(mockClient *mockedclient.Client, latestVersion string) {
	mockClient.On("GetLatestActiveGateVersion", mock.Anything).Return(latestVersion, nil)
}
