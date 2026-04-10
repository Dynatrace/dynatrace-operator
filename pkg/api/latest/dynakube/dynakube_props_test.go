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

package dynakube

import (
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestTokens(t *testing.T) {
	testName := "test-name"
	testValue := "test-value"

	t.Run("GetTokensName returns custom token name", func(t *testing.T) {
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testName},
			Spec:       DynaKubeSpec{Tokens: testValue},
		}
		assert.Equal(t, dk.Tokens(), testValue)
	})
	t.Run("GetTokensName uses instance name as default value", func(t *testing.T) {
		dk := DynaKube{ObjectMeta: metav1.ObjectMeta{Name: testName}}
		assert.Equal(t, dk.Tokens(), testName)
	})
}

func TestIsTokenScopeVerificationAllowed(t *testing.T) {
	dk := DynaKube{
		Status: DynaKubeStatus{
			DynatraceAPI: DynatraceAPIStatus{
				LastTokenScopeRequest: metav1.Time{},
			},
		},
	}

	timeProvider := timeprovider.New().Freeze()
	tests := map[string]struct {
		lastRequestTimeDeltaMinutes int
		updateExpected              bool
		threshold                   *uint16
	}{
		"Do not update after 10 minutes using default interval": {
			lastRequestTimeDeltaMinutes: -10,
			updateExpected:              false,
			threshold:                   nil,
		},
		"Do update after 20 minutes using default interval": {
			lastRequestTimeDeltaMinutes: -20,
			updateExpected:              true,
			threshold:                   nil,
		},
		"Do not update after 3 minutes using 5m interval": {
			lastRequestTimeDeltaMinutes: -3,
			updateExpected:              false,
			threshold:                   ptr.To(uint16(5)),
		},
		"Do update after 7 minutes using 5m interval": {
			lastRequestTimeDeltaMinutes: -7,
			updateExpected:              true,
			threshold:                   ptr.To(uint16(5)),
		},
		"Do not update after 17 minutes using 20m interval": {
			lastRequestTimeDeltaMinutes: -17,
			updateExpected:              false,
			threshold:                   ptr.To(uint16(20)),
		},
		"Do update after 22 minutes using 20m interval": {
			lastRequestTimeDeltaMinutes: -22,
			updateExpected:              true,
			threshold:                   ptr.To(uint16(20)),
		},
		"Do update immediately using 0m interval": {
			lastRequestTimeDeltaMinutes: 0,
			updateExpected:              true,
			threshold:                   ptr.To(uint16(0)),
		},
		"Do update after 1 minute using 0m interval": {
			lastRequestTimeDeltaMinutes: -1,
			updateExpected:              true,
			threshold:                   ptr.To(uint16(0)),
		},
		"Do update after 20 minutes using 0m interval": {
			lastRequestTimeDeltaMinutes: -20,
			updateExpected:              true,
			threshold:                   ptr.To(uint16(0)),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			dk.Spec.DynatraceAPIRequestThreshold = test.threshold

			lastRequestTime := timeProvider.Now().Add(time.Duration(test.lastRequestTimeDeltaMinutes) * time.Minute)
			dk.Status.DynatraceAPI.LastTokenScopeRequest.Time = lastRequestTime

			assert.Equal(t, test.updateExpected, dk.IsTokenScopeVerificationAllowed(timeProvider))
		})
	}
}

func TestImagePullSecretReferences(t *testing.T) {
	const (
		dkName           = "test-dk"
		customPullSecret = "my-custom-secret"
		helmPullSecret   = "helm-pull-secret"
	)

	t.Run("only tenant pull secret when no custom pull secret is set", func(t *testing.T) {
		t.Setenv(k8senv.DTOperatorPullSecretEnvName, "")
		dk := DynaKube{ObjectMeta: metav1.ObjectMeta{Name: dkName}}
		refs := dk.ImagePullSecretReferences()
		assert.Len(t, refs, 1)
		assert.Equal(t, dk.TenantRegistryPullSecretName(), refs[0].Name)
	})

	t.Run("includes DynaKube customPullSecret when set", func(t *testing.T) {
		t.Setenv(k8senv.DTOperatorPullSecretEnvName, "")
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: dkName},
			Spec:       DynaKubeSpec{CustomPullSecret: customPullSecret},
		}
		refs := dk.ImagePullSecretReferences()
		assert.Len(t, refs, 2)
		assert.Equal(t, dk.TenantRegistryPullSecretName(), refs[0].Name)
		assert.Equal(t, customPullSecret, refs[1].Name)
	})

	t.Run("includes Helm pull secret from env var", func(t *testing.T) {
		t.Setenv(k8senv.DTOperatorPullSecretEnvName, helmPullSecret)
		dk := DynaKube{ObjectMeta: metav1.ObjectMeta{Name: dkName}}
		refs := dk.ImagePullSecretReferences()
		assert.Len(t, refs, 2)
		assert.Equal(t, dk.TenantRegistryPullSecretName(), refs[0].Name)
		assert.Equal(t, helmPullSecret, refs[1].Name)
	})

	t.Run("does not duplicate helm pull secret when it matches DynaKube customPullSecret", func(t *testing.T) {
		t.Setenv(k8senv.DTOperatorPullSecretEnvName, customPullSecret)
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: dkName},
			Spec:       DynaKubeSpec{CustomPullSecret: customPullSecret},
		}
		refs := dk.ImagePullSecretReferences()
		assert.Len(t, refs, 2)
		assert.Equal(t, dk.TenantRegistryPullSecretName(), refs[0].Name)
		assert.Equal(t, customPullSecret, refs[1].Name)
	})

	t.Run("includes both DynaKube customPullSecret and helm pull secret", func(t *testing.T) {
		t.Setenv(k8senv.DTOperatorPullSecretEnvName, helmPullSecret)
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: dkName},
			Spec:       DynaKubeSpec{CustomPullSecret: customPullSecret},
		}
		refs := dk.ImagePullSecretReferences()
		assert.Len(t, refs, 3)
		assert.Equal(t, dk.TenantRegistryPullSecretName(), refs[0].Name)
		assert.Equal(t, customPullSecret, refs[1].Name)
		assert.Equal(t, helmPullSecret, refs[2].Name)
	})
}

func TestTenantRegistryPullSecretReferences(t *testing.T) {
	const dkName = "test-dk"

	t.Run("always returns only the tenant registry pull secret", func(t *testing.T) {
		dk := DynaKube{ObjectMeta: metav1.ObjectMeta{Name: dkName}}
		refs := dk.TenantRegistryPullSecretReferences()
		assert.Len(t, refs, 1)
		assert.Equal(t, dk.TenantRegistryPullSecretName(), refs[0].Name)
	})

	t.Run("does not include customPullSecret even when set", func(t *testing.T) {
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: dkName},
			Spec:       DynaKubeSpec{CustomPullSecret: "my-custom-secret"},
		}
		refs := dk.TenantRegistryPullSecretReferences()
		assert.Len(t, refs, 1)
		assert.Equal(t, dk.TenantRegistryPullSecretName(), refs[0].Name)
	})

	t.Run("does not include Helm pull secret even when set", func(t *testing.T) {
		t.Setenv(k8senv.DTOperatorPullSecretEnvName, "helm-pull-secret")
		dk := DynaKube{ObjectMeta: metav1.ObjectMeta{Name: dkName}}
		refs := dk.TenantRegistryPullSecretReferences()
		assert.Len(t, refs, 1)
		assert.Equal(t, dk.TenantRegistryPullSecretName(), refs[0].Name)
	})
}

func TestCustomPullSecretReferences(t *testing.T) {
	const (
		dkName           = "test-dk"
		customPullSecret = "my-custom-secret"
		helmPullSecret   = "helm-pull-secret"
	)

	t.Run("empty when no custom or Helm pull secret is set", func(t *testing.T) {
		t.Setenv(k8senv.DTOperatorPullSecretEnvName, "")
		dk := DynaKube{ObjectMeta: metav1.ObjectMeta{Name: dkName}}
		refs := dk.CustomPullSecretReferences()
		assert.Empty(t, refs)
	})

	t.Run("includes customPullSecret when set", func(t *testing.T) {
		t.Setenv(k8senv.DTOperatorPullSecretEnvName, "")
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: dkName},
			Spec:       DynaKubeSpec{CustomPullSecret: customPullSecret},
		}
		refs := dk.CustomPullSecretReferences()
		assert.Len(t, refs, 1)
		assert.Equal(t, customPullSecret, refs[0].Name)
	})

	t.Run("includes Helm pull secret from env var", func(t *testing.T) {
		t.Setenv(k8senv.DTOperatorPullSecretEnvName, helmPullSecret)
		dk := DynaKube{ObjectMeta: metav1.ObjectMeta{Name: dkName}}
		refs := dk.CustomPullSecretReferences()
		assert.Len(t, refs, 1)
		assert.Equal(t, helmPullSecret, refs[0].Name)
	})

	t.Run("does not duplicate when Helm pull secret matches customPullSecret", func(t *testing.T) {
		t.Setenv(k8senv.DTOperatorPullSecretEnvName, customPullSecret)
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: dkName},
			Spec:       DynaKubeSpec{CustomPullSecret: customPullSecret},
		}
		refs := dk.CustomPullSecretReferences()
		assert.Len(t, refs, 1)
		assert.Equal(t, customPullSecret, refs[0].Name)
	})

	t.Run("includes both customPullSecret and Helm pull secret", func(t *testing.T) {
		t.Setenv(k8senv.DTOperatorPullSecretEnvName, helmPullSecret)
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: dkName},
			Spec:       DynaKubeSpec{CustomPullSecret: customPullSecret},
		}
		refs := dk.CustomPullSecretReferences()
		assert.Len(t, refs, 2)
		assert.Equal(t, customPullSecret, refs[0].Name)
		assert.Equal(t, helmPullSecret, refs[1].Name)
	})

	t.Run("never contains the operator-generated tenant registry pull secret", func(t *testing.T) {
		t.Setenv(k8senv.DTOperatorPullSecretEnvName, helmPullSecret)
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: dkName},
			Spec:       DynaKubeSpec{CustomPullSecret: customPullSecret},
		}
		refs := dk.CustomPullSecretReferences()
		assert.Len(t, refs, 2)
		for _, ref := range refs {
			assert.NotEqual(t, dk.TenantRegistryPullSecretName(), ref.Name)
		}
	})
}
