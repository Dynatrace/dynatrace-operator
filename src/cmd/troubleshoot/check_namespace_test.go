package troubleshoot

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testDynakube       = "dynakube"
	testOtherDynakube  = "otherdynakube"
	testUID            = "test-uid"
	testNamespace      = "dynatrace"
	testOtherNamespace = "othernamespace"
)

func TestTroubleshootNamespace(t *testing.T) {
	t.Run("namespace exists in cluster", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: testNamespace,
					UID:  testUID,
				},
			}).
			Build()

		assert.NoErrorf(t, checkNamespace(clt, &troubleshootContext), "'%s' namespace not found", troubleshootContext.namespaceName)
	})
	t.Run("namespace does not exist in cluster", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testNamespace, dynakubeName: testDynakube}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: testOtherNamespace,
					UID:  testUID,
				},
			}).
			Build()

		assert.Errorf(t, checkNamespace(clt, &troubleshootContext), "'%s' namespace found", troubleshootContext.namespaceName)
	})
	t.Run("invalid namespace selected", func(t *testing.T) {
		troubleshootContext := TestData{namespaceName: testOtherNamespace, dynakubeName: testDynakube}

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: testNamespace,
					UID:  testUID,
				},
			}).
			Build()

		assert.Errorf(t, checkNamespace(clt, &troubleshootContext), "'%s' namespace found", troubleshootContext.namespaceName)
	})
}
