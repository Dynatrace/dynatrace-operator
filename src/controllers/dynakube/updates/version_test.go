/*
Copyright 2021 Dynatrace LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package updates

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dtversion"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/status"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
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

	dkTemplate := dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: testName, Namespace: testNamespace},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: testApiUrl,
			KubernetesMonitoring: dynatracev1beta1.KubernetesMonitoringSpec{
				Enabled: true,
			},
			OneAgent: dynatracev1beta1.OneAgentSpec{
				ClassicFullStack: &dynatracev1beta1.HostInjectSpec{},
			},
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				Capabilities: []dynatracev1beta1.CapabilityDisplayName{
					dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.StatsdIngestCapability.ShortName),
				},
			},
		},
	}

	t.Run("no update if version provider returns error", func(t *testing.T) {
		dk, fakeClient, now, registry := dkTemplate.DeepCopy(), fake.NewClient(), metav1.Now(), newEmptyFakeRegistry()
		dkState := &status.DynakubeState{Instance: dk, Now: now}

		upd, err := ReconcileVersions(ctx, dkState, fakeClient, registry.ImageVersionExt)

		assert.Error(t, err)
		assert.False(t, upd)
	})

	t.Run("all image versions were updated", func(t *testing.T) {
		dk := dkTemplate.DeepCopy()

		dkState, fakeClient, now := testInitDynakubeState(t, dk)
		status := &dkState.Instance.Status
		registry := newFakeRegistry(map[string]string{
			agImagePath:       "1.0.0",
			eecImagePath:      "1.0.0",
			statsdImagePath:   "1.0.0",
			oneAgentImagePath: "1.0.0",
		})

		{
			upd, err := ReconcileVersions(ctx, dkState, fakeClient, registry.ImageVersionExt)
			assert.NoError(t, err)
			assert.True(t, upd)
			assertVersionStatusEquals(t, registry, agImagePath, now, &status.ActiveGate)
			assertVersionStatusEquals(t, registry, oneAgentImagePath, now, &status.OneAgent)
			assertVersionStatusEquals(t, registry, eecImagePath, now, &status.ExtensionController)
			assertVersionStatusEquals(t, registry, statsdImagePath, now, &status.Statsd)
		}
		{
			upd, err := ReconcileVersions(ctx, dkState, fakeClient, registry.ImageVersionExt)
			assert.NoError(t, err)
			assert.False(t, upd)
		}
	})

	t.Run("some image versions were updated", func(t *testing.T) {
		dk := dkTemplate.DeepCopy()
		dkState, fakeClient, now := testInitDynakubeState(t, dk)
		status := &dkState.Instance.Status
		registry := newFakeRegistry(map[string]string{
			agImagePath:       "1.0.0",
			eecImagePath:      "1.0.0",
			statsdImagePath:   "1.0.0",
			oneAgentImagePath: "1.0.0",
		})

		{
			upd, err := ReconcileVersions(ctx, dkState, fakeClient, registry.ImageVersionExt)
			assert.NoError(t, err)
			assert.True(t, upd)

			assertVersionStatusEquals(t, registry, agImagePath, now, &status.ActiveGate)
			assertVersionStatusEquals(t, registry, oneAgentImagePath, now, &status.OneAgent)
			assertVersionStatusEquals(t, registry, eecImagePath, now, &status.ExtensionController)
			assertVersionStatusEquals(t, registry, statsdImagePath, now, &status.Statsd)
		}

		registry.SetVersion(eecImagePath, "1.0.1")
		{
			upd, err := ReconcileVersions(ctx, dkState, fakeClient, registry.ImageVersionExt)
			assert.NoError(t, err)
			assert.False(t, upd)

			assertVersionStatusEquals(t, registry, agImagePath, now, &status.ActiveGate)
			assertVersionStatusEquals(t, registry, oneAgentImagePath, now, &status.OneAgent)
			assertVersionStatusEquals(t, newFakeRegistry(map[string]string{
				eecImagePath: "1.0.0", // Previous state
			}), eecImagePath, now, &status.ExtensionController)
			assertVersionStatusEquals(t, registry, statsdImagePath, now, &status.Statsd)
		}

		now = testChangeTime(t, dkState, 15*time.Minute+1*time.Second)
		{
			upd, err := ReconcileVersions(ctx, dkState, fakeClient, registry.ImageVersionExt)
			assert.NoError(t, err)
			assert.True(t, upd)

			assertVersionStatusEquals(t, registry, agImagePath, now, &status.ActiveGate)
			assertVersionStatusEquals(t, registry, oneAgentImagePath, now, &status.OneAgent)
			assertVersionStatusEquals(t, registry, eecImagePath, now, &status.ExtensionController)
			assertVersionStatusEquals(t, registry, statsdImagePath, now, &status.Statsd)
		}
	})
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

func (registry *fakeRegistry) ImageVersion(imagePath string) (dtversion.ImageVersion, error) {
	if version, exists := registry.imageVersions[imagePath]; !exists {
		return dtversion.ImageVersion{}, fmt.Errorf(`cannot provide version for image: "%s"`, imagePath)
	} else {
		return dtversion.ImageVersion{
			Version: version,
			Hash:    fmt.Sprintf("%x", sha256.Sum256([]byte(imagePath+":"+version))),
		}, nil
	}
}

func (registry *fakeRegistry) ImageVersionExt(imagePath string, _ *dtversion.DockerConfig) (dtversion.ImageVersion, error) {
	return registry.ImageVersion(imagePath)
}

func assertVersionStatusEquals(t *testing.T, registry *fakeRegistry, imagePath string, timePoint metav1.Time, versionStatusNamer dynatracev1beta1.VersionStatusNamer) {
	expectedVersion, err := registry.ImageVersion(imagePath)

	assert.NoError(t, err, "Image version is unexpectedly unknown for '%s'", imagePath)
	assert.Equalf(t, expectedVersion.Version, versionStatusNamer.Status().Version, "Unexpected version for versioned component %s", versionStatusNamer.Name())
	assert.Equalf(t, expectedVersion.Hash, versionStatusNamer.Status().ImageHash, "Unexpected image hash for versioned component %s", versionStatusNamer.Name())
	if ts := versionStatusNamer.Status().LastUpdateProbeTimestamp; assert.NotNilf(t, ts, "Unexpectedly missing update timestamp for versioned component %s", versionStatusNamer.Name()) {
		assert.Equalf(t, timePoint, *ts, "Unexpected update timestamp for versioned component %s", versionStatusNamer.Name())
	}
}

func testInitDynakubeState(t *testing.T, dk *dynatracev1beta1.DynaKube) (*status.DynakubeState, client.Client, metav1.Time) {
	fakeClient := fake.NewClient()
	now := metav1.Now()
	dkState := &status.DynakubeState{Instance: dk, Now: now}

	data := func() []byte {
		data, err := buildTestDockerAuth(t)
		require.NoError(t, err)
		return data
	}()
	{
		err := createTestPullSecret(t, fakeClient, dkState, data)
		require.NoError(t, err)
	}

	return dkState, fakeClient, now
}

func testChangeTime(_ *testing.T, dkState *status.DynakubeState, duration time.Duration) metav1.Time {
	dkState.Now = metav1.NewTime(dkState.Now.Add(duration))
	return dkState.Now
}

// Adding *testing.T parameter to prevent usage in production code
func createTestPullSecret(_ *testing.T, clt client.Client, dkState *status.DynakubeState, data []byte) error {
	return clt.Create(context.TODO(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: dkState.Instance.Namespace,
			Name:      dkState.Instance.Name + dtpullsecret.PullSecretSuffix,
		},
		Data: map[string][]byte{
			".dockerconfigjson": data,
		},
	})
}

// Adding *testing.T parameter to prevent usage in production code
func buildTestDockerAuth(_ *testing.T) ([]byte, error) {
	dockerConf := struct {
		Auths map[string]dtversion.DockerAuth `json:"auths"`
	}{
		Auths: map[string]dtversion.DockerAuth{
			testRegistry: {
				Username: testName,
				Password: testPaaSToken,
			},
		},
	}
	return json.Marshal(dockerConf)
}
