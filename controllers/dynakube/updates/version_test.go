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
	"encoding/json"
	"errors"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtversion"
	"github.com/Dynatrace/dynatrace-operator/controllers/utils"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testName      = "test-name"
	testNamespace = "test-namespace"
	testPaaSToken = "test-paas-token"
	testRegistry  = "registry"
	testVersion   = "1.0.0"
	testHash      = "abcdefg1234"
)

func TestReconcile_UpdateImageVersion(t *testing.T) {
	ctx := context.Background()

	dk := dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: testName, Namespace: testNamespace},
		Spec: dynatracev1alpha1.DynaKubeSpec{
			KubernetesMonitoringSpec: dynatracev1alpha1.KubernetesMonitoringSpec{
				CapabilityProperties: dynatracev1alpha1.CapabilityProperties{Enabled: true},
			},
			ClassicFullStack: dynatracev1alpha1.FullStackSpec{Enabled: true, UseImmutableImage: true},
		},
	}

	fakeClient := fake.NewClient()

	now := metav1.Now()
	rec := &utils.Reconciliation{Instance: &dk, Log: logger.NewDTLogger(), Now: now}

	errVerProvider := func(img string, dockerConfig *dtversion.DockerConfig) (dtversion.ImageVersion, error) {
		return dtversion.ImageVersion{}, errors.New("Not implemented")
	}

	upd, err := ReconcileVersions(ctx, rec, fakeClient, nil, errVerProvider)
	assert.Error(t, err)
	assert.False(t, upd)

	data, err := buildTestDockerAuth(t)
	require.NoError(t, err)

	err = createTestPullSecret(t, fakeClient, rec, data)
	require.NoError(t, err)

	sampleVerProvider := func(img string, dockerConfig *dtversion.DockerConfig) (dtversion.ImageVersion, error) {
		return dtversion.ImageVersion{Version: testVersion, Hash: testHash}, nil
	}

	upd, err = ReconcileVersions(ctx, rec, fakeClient, nil, sampleVerProvider)
	assert.NoError(t, err)
	assert.True(t, upd)

	assert.Equal(t, testVersion, rec.Instance.Status.ActiveGate.Version)
	assert.Equal(t, testHash, rec.Instance.Status.ActiveGate.ImageHash)
	if ts := rec.Instance.Status.ActiveGate.LastUpdateProbeTimestamp; assert.NotNil(t, ts) {
		assert.Equal(t, now, *ts)
	}

	assert.Equal(t, testVersion, rec.Instance.Status.OneAgent.Version)
	assert.Equal(t, testHash, rec.Instance.Status.OneAgent.ImageHash)
	if ts := rec.Instance.Status.OneAgent.LastUpdateProbeTimestamp; assert.NotNil(t, ts) {
		assert.Equal(t, now, *ts)
	}

	upd, err = ReconcileVersions(ctx, rec, fakeClient, nil, sampleVerProvider)
	assert.NoError(t, err)
	assert.False(t, upd)
}

// Adding *testing.T parameter to prevent usage in production code
func createTestPullSecret(_ *testing.T, clt client.Client, rec *utils.Reconciliation, data []byte) error {
	return clt.Create(context.TODO(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: rec.Instance.Namespace,
			Name:      rec.Instance.Name + dtpullsecret.PullSecretSuffix,
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
