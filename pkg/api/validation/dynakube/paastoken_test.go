package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dttoken"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeprecatedPaasToken(t *testing.T) {
	t.Run("gen2 token without paasToken", func(t *testing.T) {
		assertAllowedWithoutWarnings(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultDynakubeObjectMeta.Name,
					Namespace: defaultDynakubeObjectMeta.Namespace,
				},
				Data: map[string][]byte{
					token.APIKey: []byte("test-api-token"),
				},
				Type: corev1.SecretTypeOpaque,
			})
	})
	t.Run("platform token without paasToken", func(t *testing.T) {
		assertAllowedWithoutWarnings(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultDynakubeObjectMeta.Name,
					Namespace: defaultDynakubeObjectMeta.Namespace,
				},
				Data: map[string][]byte{
					token.APIKey: []byte(dttoken.PlatformPrefix + "test-platform-token"),
				},
				Type: corev1.SecretTypeOpaque,
			})
	})
	t.Run("gen2 token with paasToken", func(t *testing.T) {
		assertAllowedWithWarnings(t, 1,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultDynakubeObjectMeta.Name,
					Namespace: defaultDynakubeObjectMeta.Namespace,
				},
				Data: map[string][]byte{
					token.APIKey:  []byte("test-api-token"),
					token.PaaSKey: []byte("test-paas-token"),
				},
				Type: corev1.SecretTypeOpaque,
			})
	})
	t.Run("platform token with paasToken", func(t *testing.T) {
		assertAllowedWithWarnings(t, 1,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultDynakubeObjectMeta.Name,
					Namespace: defaultDynakubeObjectMeta.Namespace,
				},
				Data: map[string][]byte{
					token.APIKey:  []byte(dttoken.PlatformPrefix + "test-platform-token"),
					token.PaaSKey: []byte("test-paas-token"),
				},
				Type: corev1.SecretTypeOpaque,
			})
	})
}
