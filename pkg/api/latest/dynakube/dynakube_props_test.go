// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dynakube

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testDKName           = "test-dk"
	testCustomPullSecret = "my-custom-secret"
	testHelmPullSecret   = "helm-pull-secret"
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

func TestAPIURL(t *testing.T) {
	t.Run("2nd gen URL is returned unchanged", func(t *testing.T) {
		dk := DynaKube{Spec: DynaKubeSpec{APIURL: "https://tenant.live.dynatrace.com/api"}}
		assert.Equal(t, "https://tenant.live.dynatrace.com/api", dk.APIURL())
		assert.Equal(t, "tenant.live.dynatrace.com", dk.APIURLHost())
	})

	t.Run("3rd gen URL is mapped to its 2nd gen equivalent", func(t *testing.T) {
		dk := DynaKube{Spec: DynaKubeSpec{APIURL: "https://tenant.apps.dynatrace.com"}}
		assert.Equal(t, "https://tenant.live.dynatrace.com/api", dk.APIURL())
	})

	t.Run("APIURLHost returns the raw host so a 3rd gen registry breaks on purpose", func(t *testing.T) {
		dk := DynaKube{Spec: DynaKubeSpec{APIURL: "https://tenant.apps.dynatrace.com"}}
		assert.Equal(t, "tenant.apps.dynatrace.com", dk.APIURLHost())
	})

	t.Run("Spec.APIURL keeps the raw user-provided value", func(t *testing.T) {
		dk := DynaKube{Spec: DynaKubeSpec{APIURL: "https://tenant.apps.dynatrace.com"}}
		assert.Equal(t, "https://tenant.apps.dynatrace.com", dk.Spec.APIURL)
	})
}

func TestImagePullSecretReferences(t *testing.T) {
	t.Run("only tenant pull secret when no custom pull secret is set", func(t *testing.T) {
		t.Setenv(k8senv.DTOperatorPullSecretEnvName, "")
		dk := DynaKube{ObjectMeta: metav1.ObjectMeta{Name: testDKName}}
		refs := dk.ImagePullSecretReferences()
		assert.Len(t, refs, 1)
		assert.Equal(t, dk.TenantRegistryPullSecretName(), refs[0].Name)
	})

	t.Run("includes DynaKube customPullSecret when set", func(t *testing.T) {
		t.Setenv(k8senv.DTOperatorPullSecretEnvName, "")
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testDKName},
			Spec:       DynaKubeSpec{CustomPullSecret: testCustomPullSecret},
		}
		refs := dk.ImagePullSecretReferences()
		assert.Len(t, refs, 2)
		assert.Equal(t, dk.TenantRegistryPullSecretName(), refs[0].Name)
		assert.Equal(t, testCustomPullSecret, refs[1].Name)
	})

	t.Run("includes Helm pull secret from env var", func(t *testing.T) {
		t.Setenv(k8senv.DTOperatorPullSecretEnvName, testHelmPullSecret)
		dk := DynaKube{ObjectMeta: metav1.ObjectMeta{Name: testDKName}}
		refs := dk.ImagePullSecretReferences()
		assert.Len(t, refs, 2)
		assert.Equal(t, dk.TenantRegistryPullSecretName(), refs[0].Name)
		assert.Equal(t, testHelmPullSecret, refs[1].Name)
	})

	t.Run("does not duplicate helm pull secret when it matches DynaKube customPullSecret", func(t *testing.T) {
		t.Setenv(k8senv.DTOperatorPullSecretEnvName, testCustomPullSecret)
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testDKName},
			Spec:       DynaKubeSpec{CustomPullSecret: testCustomPullSecret},
		}
		refs := dk.ImagePullSecretReferences()
		assert.Len(t, refs, 2)
		assert.Equal(t, dk.TenantRegistryPullSecretName(), refs[0].Name)
		assert.Equal(t, testCustomPullSecret, refs[1].Name)
	})

	t.Run("includes both DynaKube customPullSecret and helm pull secret", func(t *testing.T) {
		t.Setenv(k8senv.DTOperatorPullSecretEnvName, testHelmPullSecret)
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testDKName},
			Spec:       DynaKubeSpec{CustomPullSecret: testCustomPullSecret},
		}
		refs := dk.ImagePullSecretReferences()
		assert.Len(t, refs, 3)
		assert.Equal(t, dk.TenantRegistryPullSecretName(), refs[0].Name)
		assert.Equal(t, testCustomPullSecret, refs[1].Name)
		assert.Equal(t, testHelmPullSecret, refs[2].Name)
	})
	t.Run("don't return tenant pull secret if platform token", func(t *testing.T) {
		t.Setenv(k8senv.DTOperatorPullSecretEnvName, "")
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testDKName},
			Status:     DynaKubeStatus{APIToken: APITokenStatus{Platform: new(true)}},
		}
		refs := dk.ImagePullSecretReferences()
		assert.Empty(t, refs)
	})
	t.Run("includes DynaKube customPullSecret if platform token", func(t *testing.T) {
		t.Setenv(k8senv.DTOperatorPullSecretEnvName, "")
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testDKName},
			Spec:       DynaKubeSpec{CustomPullSecret: testCustomPullSecret},
			Status:     DynaKubeStatus{APIToken: APITokenStatus{Platform: new(true)}},
		}
		refs := dk.ImagePullSecretReferences()
		assert.Len(t, refs, 1)
		assert.Equal(t, testCustomPullSecret, refs[0].Name)
	})
	t.Run("don't return tenant pull secret if use-public-registry annotation with platform token", func(t *testing.T) {
		t.Setenv(k8senv.DTOperatorPullSecretEnvName, "")
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:        testDKName,
				Annotations: map[string]string{exp.UsePublicRegistryKey: "true"},
			},
			Status: DynaKubeStatus{APIToken: APITokenStatus{Platform: new(true)}},
		}
		refs := dk.ImagePullSecretReferences()
		assert.Empty(t, refs)
	})
	t.Run("don't return tenant pull secret if use-public-registry annotation without platform token", func(t *testing.T) {
		t.Setenv(k8senv.DTOperatorPullSecretEnvName, "")
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:        testDKName,
				Annotations: map[string]string{exp.UsePublicRegistryKey: "true"},
			},
		}
		refs := dk.ImagePullSecretReferences()
		assert.Empty(t, refs)
	})
}

func TestPullSecretNames(t *testing.T) {
	t.Run("includes tenant pull secret name", func(t *testing.T) {
		dk := DynaKube{ObjectMeta: metav1.ObjectMeta{Name: testDKName}}
		names := dk.PullSecretNames()
		assert.Contains(t, names, dk.TenantRegistryPullSecretName())
	})
	t.Run("don't return tenant pull secret name if platform token", func(t *testing.T) {
		t.Setenv(k8senv.DTOperatorPullSecretEnvName, "")
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testDKName},
			Status:     DynaKubeStatus{APIToken: APITokenStatus{Platform: new(true)}},
		}
		names := dk.PullSecretNames()
		assert.Empty(t, names)
	})
	t.Run("includes DynaKube customPullSecret name if platform token", func(t *testing.T) {
		t.Setenv(k8senv.DTOperatorPullSecretEnvName, "")
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testDKName},
			Spec:       DynaKubeSpec{CustomPullSecret: testCustomPullSecret},
			Status:     DynaKubeStatus{APIToken: APITokenStatus{Platform: new(true)}},
		}
		names := dk.PullSecretNames()
		assert.Len(t, names, 1)
		assert.Equal(t, testCustomPullSecret, names[0])
	})
}

func TestTenantRegistryPullSecretReferences(t *testing.T) {
	t.Run("always returns only the tenant registry pull secret", func(t *testing.T) {
		dk := DynaKube{ObjectMeta: metav1.ObjectMeta{Name: testDKName}}
		refs := dk.TenantRegistryPullSecretReferences()
		assert.Len(t, refs, 1)
		assert.Equal(t, dk.TenantRegistryPullSecretName(), refs[0].Name)
	})

	t.Run("does not include customPullSecret even when set", func(t *testing.T) {
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testDKName},
			Spec:       DynaKubeSpec{CustomPullSecret: testCustomPullSecret},
		}
		refs := dk.TenantRegistryPullSecretReferences()
		assert.Len(t, refs, 1)
		assert.Equal(t, dk.TenantRegistryPullSecretName(), refs[0].Name)
	})

	t.Run("does not include Helm pull secret even when set", func(t *testing.T) {
		t.Setenv(k8senv.DTOperatorPullSecretEnvName, testHelmPullSecret)
		dk := DynaKube{ObjectMeta: metav1.ObjectMeta{Name: testDKName}}
		refs := dk.TenantRegistryPullSecretReferences()
		assert.Len(t, refs, 1)
		assert.Equal(t, dk.TenantRegistryPullSecretName(), refs[0].Name)
	})
}

func TestCustomPullSecretReferences(t *testing.T) {
	t.Run("empty when no custom or Helm pull secret is set", func(t *testing.T) {
		t.Setenv(k8senv.DTOperatorPullSecretEnvName, "")
		dk := DynaKube{ObjectMeta: metav1.ObjectMeta{Name: testDKName}}
		refs := dk.CustomPullSecretReferences()
		assert.Empty(t, refs)
	})

	t.Run("includes customPullSecret when set", func(t *testing.T) {
		t.Setenv(k8senv.DTOperatorPullSecretEnvName, "")
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testDKName},
			Spec:       DynaKubeSpec{CustomPullSecret: testCustomPullSecret},
		}
		refs := dk.CustomPullSecretReferences()
		assert.Len(t, refs, 1)
		assert.Equal(t, testCustomPullSecret, refs[0].Name)
	})

	t.Run("includes Helm pull secret from env var", func(t *testing.T) {
		t.Setenv(k8senv.DTOperatorPullSecretEnvName, testHelmPullSecret)
		dk := DynaKube{ObjectMeta: metav1.ObjectMeta{Name: testDKName}}
		refs := dk.CustomPullSecretReferences()
		assert.Len(t, refs, 1)
		assert.Equal(t, testHelmPullSecret, refs[0].Name)
	})

	t.Run("does not duplicate when Helm pull secret matches customPullSecret", func(t *testing.T) {
		t.Setenv(k8senv.DTOperatorPullSecretEnvName, testCustomPullSecret)
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testDKName},
			Spec:       DynaKubeSpec{CustomPullSecret: testCustomPullSecret},
		}
		refs := dk.CustomPullSecretReferences()
		assert.Len(t, refs, 1)
		assert.Equal(t, testCustomPullSecret, refs[0].Name)
	})

	t.Run("includes both customPullSecret and Helm pull secret", func(t *testing.T) {
		t.Setenv(k8senv.DTOperatorPullSecretEnvName, testHelmPullSecret)
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testDKName},
			Spec:       DynaKubeSpec{CustomPullSecret: testCustomPullSecret},
		}
		refs := dk.CustomPullSecretReferences()
		assert.Len(t, refs, 2)
		assert.Equal(t, testCustomPullSecret, refs[0].Name)
		assert.Equal(t, testHelmPullSecret, refs[1].Name)
	})

	t.Run("never contains the operator-generated tenant registry pull secret", func(t *testing.T) {
		t.Setenv(k8senv.DTOperatorPullSecretEnvName, testHelmPullSecret)
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testDKName},
			Spec:       DynaKubeSpec{CustomPullSecret: testCustomPullSecret},
		}
		refs := dk.CustomPullSecretReferences()
		assert.Len(t, refs, 2)
		for _, ref := range refs {
			assert.NotEqual(t, dk.TenantRegistryPullSecretName(), ref.Name)
		}
	})
}
