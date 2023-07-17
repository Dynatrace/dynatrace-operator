package version

import (
	"context"
	"testing"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/src/timeprovider"
	"github.com/opencontainers/go-digest"
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
)

func TestReconcile(t *testing.T) {
	ctx := context.Background()
	latestAgentVersion := "1.2.3.4-5"
	testOneAgentHash := digest.FromString("sha256:7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f")
	testActiveGateHash := digest.FromString("sha256:7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d72f")
	testCodeModulesHash := digest.FromString("sha256:7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d82f")

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
		faultyRegistry := newEmptyFakeRegistry()

		versionReconciler := Reconciler{
			dynakube:     dynakubeTemplate.DeepCopy(),
			apiReader:    fake.NewClient(),
			fs:           afero.Afero{Fs: afero.NewMemMapFs()},
			versionFunc:  faultyRegistry.ImageVersionExt,
			timeProvider: timeprovider.New().Freeze(),
		}
		err := versionReconciler.Reconcile(ctx)
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
		registry := newFakeRegistry(map[string]ImageVersion{
			dynakube.DefaultActiveGateImage(): {
				Version: testActiveGateImage.Tag,
				Digest:  testActiveGateHash,
			},
			dynakube.DefaultOneAgentImage(): {
				Version: testOneAgentImage.Tag,
				Digest:  testOneAgentHash,
			},
		})
		mockClient := &dtclient.MockDynatraceClient{}
		mockLatestAgentVersion(mockClient, latestAgentVersion)

		versionReconciler := Reconciler{
			dynakube:     dynakube,
			apiReader:    fakeClient,
			fs:           afero.Afero{Fs: afero.NewMemMapFs()},
			versionFunc:  registry.ImageVersionExt,
			timeProvider: timeProvider,
			dtClient:     mockClient,
		}
		err := versionReconciler.Reconcile(ctx)
		require.NoError(t, err)
		assertStatusBasedOnTenantRegistry(t, dynakube.DefaultActiveGateImage(), testActiveGateImage.Tag, dkStatus.ActiveGate.VersionStatus)
		assertStatusBasedOnTenantRegistry(t, dynakube.DefaultOneAgentImage(), testOneAgentImage.Tag, dkStatus.OneAgent.VersionStatus)
		assert.Equal(t, latestAgentVersion, dkStatus.CodeModules.VersionStatus.Version)

		// no change if probe not old enough
		previousProbe := *dkStatus.CodeModules.VersionStatus.LastProbeTimestamp
		err = versionReconciler.Reconcile(ctx)
		require.NoError(t, err)
		assert.Equal(t, previousProbe, *dkStatus.CodeModules.VersionStatus.LastProbeTimestamp)

		// change if probe old enough
		changeTime(timeProvider, 15*time.Minute+1*time.Second)
		err = versionReconciler.Reconcile(ctx)
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

		registry := newFakeRegistry(map[string]ImageVersion{
			testActiveGateImage.String(): {
				Version: testActiveGateImage.Tag,
				Digest:  testActiveGateHash,
			},
			testOneAgentImage.String(): {
				Version: testOneAgentImage.Tag,
				Digest:  testOneAgentHash,
			},
			testCodeModulesImage.String(): {
				Version: testCodeModulesImage.Tag,
				Digest:  testCodeModulesHash,
			},
		})
		mockClient := &dtclient.MockDynatraceClient{}
		mockActiveGateImageInfo(mockClient, testActiveGateImage)
		mockCodeModulesImageInfo(mockClient, testCodeModulesImage)
		mockOneAgentImageInfo(mockClient, testOneAgentImage)

		versionReconciler := Reconciler{
			dynakube:     dynakube,
			apiReader:    fakeClient,
			fs:           afero.Afero{Fs: afero.NewMemMapFs()},
			versionFunc:  registry.ImageVersionExt,
			timeProvider: timeprovider.New().Freeze(),
			dtClient:     mockClient,
		}
		err := versionReconciler.Reconcile(ctx)
		require.NoError(t, err)
		assertPublicRegistryVersionStatusEquals(t, registry, getTaggedReference(t, testActiveGateImage.String()), dkStatus.ActiveGate.VersionStatus)
		assertPublicRegistryVersionStatusEquals(t, registry, getTaggedReference(t, testOneAgentImage.String()), dkStatus.OneAgent.VersionStatus)
		assertPublicRegistryVersionStatusEquals(t, registry, getTaggedReference(t, testCodeModulesImage.String()), dkStatus.CodeModules.VersionStatus)
	})
}

func TestNeedsReconcile(t *testing.T) {
	timeProvider := timeprovider.New().Freeze()

	dynakube := dynatracev1beta1.DynaKube{
		Spec: dynatracev1beta1.DynaKubeSpec{
			OneAgent: dynatracev1beta1.OneAgentSpec{
				ClassicFullStack: &dynatracev1beta1.HostInjectSpec{},
			},
		},
	}

	t.Run("return only updaters needed", func(t *testing.T) {
		reconciler := Reconciler{
			dynakube:     &dynakube,
			timeProvider: timeProvider,
		}
		updaters := []versionStatusUpdater{
			newOneAgentUpdater(&dynakube, nil, nil),
			newActiveGateUpdater(&dynakube, nil, nil),
		}

		neededUpdater := reconciler.needsReconcile(updaters)

		assert.Len(t, neededUpdater, 1)
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
				VersionStatus: dynatracev1beta1.VersionStatus{
					Source: dynatracev1beta1.TenantRegistryVersionSource,
				},
			},
		},
	}

	t.Run("needs", func(t *testing.T) {
		updatedDynakube := dynakube.DeepCopy()
		reconciler := Reconciler{
			dynakube:     updatedDynakube,
			timeProvider: timeProvider,
		}
		assert.True(t, reconciler.needsUpdate(newOneAgentUpdater(updatedDynakube, nil, nil)))
	})
	t.Run("does not need", func(t *testing.T) {
		reconciler := Reconciler{
			dynakube:     &dynatracev1beta1.DynaKube{},
			timeProvider: timeProvider,
		}
		assert.False(t, reconciler.needsUpdate(newOneAgentUpdater(&dynatracev1beta1.DynaKube{}, nil, nil)))
	})
	t.Run("does not need, because not old enough", func(t *testing.T) {
		oldImage := "repo.com:tag@sha256:123"
		newImage := "repo.com:tag"
		updatedDynakube := dynakube.DeepCopy()
		setOneAgentCustomImageStatus(updatedDynakube, oldImage)
		updatedDynakube.Spec.OneAgent.ClassicFullStack.Image = newImage
		updatedDynakube.Status.OneAgent.LastProbeTimestamp = timeProvider.Now()
		reconciler := Reconciler{
			dynakube:     updatedDynakube,
			timeProvider: timeProvider,
		}
		assert.False(t, reconciler.needsUpdate(newOneAgentUpdater(updatedDynakube, nil, nil)))
	})

	t.Run("needs, because source changed", func(t *testing.T) {
		updatedDynakube := dynakube.DeepCopy()
		setOneAgentCustomImageStatus(updatedDynakube, "")
		reconciler := Reconciler{
			dynakube:     updatedDynakube,
			timeProvider: timeProvider,
		}
		assert.True(t, reconciler.needsUpdate(newOneAgentUpdater(updatedDynakube, nil, nil)))
	})

	t.Run("needs, because custom image changed", func(t *testing.T) {
		oldImage := "repo.com:tag@sha256:123"
		newImage := "repo.com:newTag"
		updatedDynakube := dynakube.DeepCopy()
		updatedDynakube.Spec.OneAgent.ClassicFullStack.Image = newImage
		setOneAgentCustomImageStatus(updatedDynakube, oldImage)
		reconciler := Reconciler{
			dynakube:     updatedDynakube,
			timeProvider: timeProvider,
		}
		assert.True(t, reconciler.needsUpdate(newOneAgentUpdater(updatedDynakube, nil, nil)))
	})

	t.Run("needs, because custom version changed", func(t *testing.T) {
		oldVersion := "1.2.3"
		newVersion := "2.4.5"
		updatedDynakube := dynakube.DeepCopy()
		updatedDynakube.Spec.OneAgent.ClassicFullStack.Version = newVersion
		setOneAgentCustomVersionStatus(updatedDynakube, oldVersion)
		reconciler := Reconciler{
			dynakube:     updatedDynakube,
			timeProvider: timeProvider,
		}
		assert.True(t, reconciler.needsUpdate(newOneAgentUpdater(updatedDynakube, nil, nil)))
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
		oldVersion := "1.2.3"
		newVersion := "2.4.5"
		updatedDynakube := dynakube.DeepCopy()
		updatedDynakube.Spec.OneAgent.ClassicFullStack.Version = newVersion
		setOneAgentCustomVersionStatus(updatedDynakube, oldVersion)
		assert.True(t, hasCustomFieldChanged(newOneAgentUpdater(updatedDynakube, nil, nil)))
	})

	t.Run("no change; version", func(t *testing.T) {
		version := "1.2.3"
		updatedDynakube := dynakube.DeepCopy()
		updatedDynakube.Spec.OneAgent.ClassicFullStack.Version = version
		setOneAgentCustomVersionStatus(updatedDynakube, version)
		assert.False(t, hasCustomFieldChanged(newOneAgentUpdater(updatedDynakube, nil, nil)))
	})

	t.Run("image changed", func(t *testing.T) {
		oldImage := "repo.com:tag@sha256:123"
		newImage := "repo.com:Tag"
		updatedDynakube := dynakube.DeepCopy()
		updatedDynakube.Spec.OneAgent.ClassicFullStack.Image = newImage
		setOneAgentCustomImageStatus(updatedDynakube, oldImage)
		assert.True(t, hasCustomFieldChanged(newOneAgentUpdater(updatedDynakube, nil, nil)))
	})

	t.Run("no change; image", func(t *testing.T) {
		oldImage := "repo.com:tag@sha256:123"
		newImage := "repo.com:tag"
		updatedDynakube := dynakube.DeepCopy()
		updatedDynakube.Spec.OneAgent.ClassicFullStack.Version = newImage
		setOneAgentCustomImageStatus(updatedDynakube, oldImage)
		assert.False(t, hasCustomFieldChanged(newOneAgentUpdater(updatedDynakube, nil, nil)))
	})
}

func setOneAgentCustomVersionStatus(dynakube *dynatracev1beta1.DynaKube, version string) {
	dynakube.Status.OneAgent.Source = dynatracev1beta1.CustomVersionVersionSource
	dynakube.Status.OneAgent.Version = version
}

func setOneAgentCustomImageStatus(dynakube *dynatracev1beta1.DynaKube, image string) {
	dynakube.Status.OneAgent.Source = dynatracev1beta1.CustomImageVersionSource
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
		Tag:    "1.2.3.4-5",
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

func mockActiveGateImageInfo(mockClient *dtclient.MockDynatraceClient, imageInfo dtclient.LatestImageInfo) {
	mockClient.On("GetLatestActiveGateImage").Return(&imageInfo, nil)
}

func mockCodeModulesImageInfo(mockClient *dtclient.MockDynatraceClient, imageInfo dtclient.LatestImageInfo) {
	mockClient.On("GetLatestCodeModulesImage").Return(&imageInfo, nil)
}

func mockOneAgentImageInfo(mockClient *dtclient.MockDynatraceClient, imageInfo dtclient.LatestImageInfo) {
	mockClient.On("GetLatestOneAgentImage").Return(&imageInfo, nil)
}

func mockLatestAgentVersion(mockClient *dtclient.MockDynatraceClient, latestVersion string) {
	mockClient.On("GetLatestAgentVersion", mock.Anything, mock.Anything).Return(latestVersion, nil)
}
