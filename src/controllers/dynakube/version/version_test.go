package version

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testName           = "test-name"
	testNamespace      = "test-namespace"
	testPaaSToken      = "test-paas-token"
	testRegistry       = "registry"
	testDockerRegistry = "ENVIRONMENTID.live.dynatrace.com"
	testApiUrl         = "https://" + testDockerRegistry + "/api"

	agImagePath       = testDockerRegistry + "/linux/activegate:latest"
	eecImagePath      = testDockerRegistry + "/linux/dynatrace-eec:latest"
	statsdImagePath   = testDockerRegistry + "/linux/dynatrace-datasource-statsd:latest"
	oneAgentImagePath = testDockerRegistry + "/linux/oneagent:latest"
)

func TestReconcile_UpdateImageVersion(t *testing.T) {
	ctx := context.Background()

	dynakubeTemplate := dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: testName, Namespace: testNamespace},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: testApiUrl,
			OneAgent: dynatracev1beta1.OneAgentSpec{
				ClassicFullStack: &dynatracev1beta1.HostInjectSpec{},
			},
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				Capabilities: []dynatracev1beta1.CapabilityDisplayName{
					dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.KubeMonCapability.ShortName),
					dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.StatsdIngestCapability.ShortName),
				},
			},
		},
	}

	t.Run("no update if version provider returns error", func(t *testing.T) {
		dynakube := dynakubeTemplate.DeepCopy()
		fakeClient := fake.NewClient()
		timeProvider := kubeobjects.NewTimeProvider()
		registry := newEmptyFakeRegistry()
		fs := afero.Afero{Fs: afero.NewMemMapFs()}

		err := ReconcileVersions(ctx, dynakube, fakeClient, fs, registry.ImageVersionExt, *timeProvider)
		assert.Error(t, err)
	})

	t.Run("all image versions were updated", func(t *testing.T) {
		dynakube := dynakubeTemplate.DeepCopy()
		fs := afero.Afero{Fs: afero.NewMemMapFs()}
		fakeClient := fake.NewClient()
		timeProvider := kubeobjects.NewTimeProvider()
		setupPullSecret(t, fakeClient, *dynakube)

		dkStatus := &dynakube.Status
		registry := newFakeRegistry(map[string]string{
			agImagePath:       "1.0.0",
			eecImagePath:      "1.0.0",
			statsdImagePath:   "1.0.0",
			oneAgentImagePath: "1.0.0",
		})

		err := ReconcileVersions(ctx, dynakube, fakeClient, fs, registry.ImageVersionExt, *timeProvider)
		assert.NoError(t, err)
		assertVersionStatusEquals(t, registry, agImagePath, *timeProvider, &dkStatus.ActiveGate)
		assertVersionStatusEquals(t, registry, oneAgentImagePath, *timeProvider, &dkStatus.OneAgent)
		assertVersionStatusEquals(t, registry, eecImagePath, *timeProvider, &dkStatus.ExtensionController)
		assertVersionStatusEquals(t, registry, statsdImagePath, *timeProvider, &dkStatus.Statsd)

		err = ReconcileVersions(ctx, dynakube, fakeClient, fs, registry.ImageVersionExt, *timeProvider)
		assert.NoError(t, err)

	})

	t.Run("some image versions were updated", func(t *testing.T) {
		dynakube := dynakubeTemplate.DeepCopy()
		fakeClient := fake.NewClient()
		timeProvider := kubeobjects.NewTimeProvider()
		setupPullSecret(t, fakeClient, *dynakube)

		fs := afero.Afero{Fs: afero.NewMemMapFs()}
		dkStatus := &dynakube.Status
		registry := newFakeRegistry(map[string]string{
			agImagePath:       "1.0.0",
			eecImagePath:      "1.0.0",
			statsdImagePath:   "1.0.0",
			oneAgentImagePath: "1.0.0",
		})

		err := ReconcileVersions(ctx, dynakube, fakeClient, fs, registry.ImageVersionExt, *timeProvider)
		assert.NoError(t, err)

		assertVersionStatusEquals(t, registry, agImagePath, *timeProvider, &dkStatus.ActiveGate)
		assertVersionStatusEquals(t, registry, oneAgentImagePath, *timeProvider, &dkStatus.OneAgent)
		assertVersionStatusEquals(t, registry, eecImagePath, *timeProvider, &dkStatus.ExtensionController)
		assertVersionStatusEquals(t, registry, statsdImagePath, *timeProvider, &dkStatus.Statsd)

		registry.SetVersion(eecImagePath, "1.0.1")

		err = ReconcileVersions(ctx, dynakube, fakeClient, fs, registry.ImageVersionExt, *timeProvider)
		assert.NoError(t, err)

		assertVersionStatusEquals(t, registry, agImagePath, *timeProvider, &dkStatus.ActiveGate)
		assertVersionStatusEquals(t, registry, oneAgentImagePath, *timeProvider, &dkStatus.OneAgent)
		assertVersionStatusEquals(t, newFakeRegistry(map[string]string{
			eecImagePath: "1.0.0", // Previous state
		}), eecImagePath, *timeProvider, &dkStatus.ExtensionController)
		assertVersionStatusEquals(t, registry, statsdImagePath, *timeProvider, &dkStatus.Statsd)

		changeTime(t, timeProvider, 15*time.Minute+1*time.Second)
		err = ReconcileVersions(ctx, dynakube, fakeClient, fs, registry.ImageVersionExt, *timeProvider)
		assert.NoError(t, err)

		assertVersionStatusEquals(t, registry, agImagePath, *timeProvider, &dkStatus.ActiveGate)
		assertVersionStatusEquals(t, registry, oneAgentImagePath, *timeProvider, &dkStatus.OneAgent)
		assertVersionStatusEquals(t, registry, eecImagePath, *timeProvider, &dkStatus.ExtensionController)
		assertVersionStatusEquals(t, registry, statsdImagePath, *timeProvider, &dkStatus.Statsd)
	})
}

func setupPullSecret(t *testing.T, fakeClient client.Client, dynakube dynatracev1beta1.DynaKube) {
	data, err := buildTestDockerAuth()
	require.NoError(t, err)
	err = createTestPullSecret(fakeClient, dynakube, data)
	require.NoError(t, err)
}

type fakeRegistry struct {
	imageVersions map[string]string
}

func newEmptyFakeRegistry() *fakeRegistry {
	return newFakeRegistry(make(map[string]string))
}

func newFakeRegistry(src map[string]string) *fakeRegistry {
	reg := fakeRegistry{
		imageVersions: make(map[string]string),
	}
	for key, val := range src {
		reg.SetVersion(key, val)
	}
	return &reg
}

func (registry *fakeRegistry) SetVersion(imagePath, version string) *fakeRegistry {
	registry.imageVersions[imagePath] = version
	return registry
}

func (registry *fakeRegistry) ImageVersion(imagePath string) (ImageVersion, error) {
	if version, exists := registry.imageVersions[imagePath]; !exists {
		return ImageVersion{}, fmt.Errorf(`cannot provide version for image: "%s"`, imagePath)
	} else {
		return ImageVersion{
			Version: version,
			Hash:    fmt.Sprintf("%x", sha256.Sum256([]byte(imagePath+":"+version))),
		}, nil
	}
}

func (registry *fakeRegistry) ImageVersionExt(imagePath string, _ *dockerconfig.DockerConfig) (ImageVersion, error) {
	return registry.ImageVersion(imagePath)
}

func assertVersionStatusEquals(t *testing.T, registry *fakeRegistry, imagePath string, timeProvider kubeobjects.TimeProvider, versionStatusNamer dynatracev1beta1.VersionStatusNamer) {
	expectedVersion, err := registry.ImageVersion(imagePath)

	assert.NoError(t, err, "Image version is unexpectedly unknown for '%s'", imagePath)
	assert.Equalf(t, expectedVersion.Version, versionStatusNamer.Status().Version, "Unexpected version for versioned component %s", versionStatusNamer.Name())
	assert.Equalf(t, expectedVersion.Hash, versionStatusNamer.Status().ImageHash, "Unexpected image hash for versioned component %s", versionStatusNamer.Name())
	if ts := versionStatusNamer.Status().LastUpdateProbeTimestamp; assert.NotNilf(t, ts, "Unexpectedly missing update timestamp for versioned component %s", versionStatusNamer.Name()) {
		assert.Equalf(t, *timeProvider.Now(), *ts, "Unexpected update timestamp for versioned component %s", versionStatusNamer.Name())
	}
}

func changeTime(_ *testing.T, timeProvider *kubeobjects.TimeProvider, duration time.Duration) {
	newTime := metav1.NewTime(timeProvider.Now().Add(duration))
	timeProvider.SetNow(&newTime)
}

func createTestPullSecret(fakeClient client.Client, dynakube dynatracev1beta1.DynaKube, data []byte) error {
	return fakeClient.Create(context.TODO(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: dynakube.Namespace,
			Name:      dynakube.Name + dtpullsecret.PullSecretSuffix,
		},
		Data: map[string][]byte{
			".dockerconfigjson": data,
		},
	})
}

func buildTestDockerAuth() ([]byte, error) {
	dockerConf := struct {
		Auths map[string]dockerconfig.DockerAuth `json:"auths"`
	}{
		Auths: map[string]dockerconfig.DockerAuth{
			testRegistry: {
				Username: testName,
				Password: testPaaSToken,
			},
		},
	}
	return json.Marshal(dockerConf)
}
