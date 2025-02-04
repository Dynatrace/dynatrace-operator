package troubleshoot

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testDynakube       = "dynakube"
	testUID            = "test-uid"
	testNamespace      = "dynatrace"
	testOtherNamespace = "othernamespace"
)

func TestTroubleshootNamespace(t *testing.T) {
	t.Run("namespace exists in cluster", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: testNamespace,
					UID:  testUID,
				},
			}).
			Build()

		require.NoErrorf(t, checkNamespace(context.Background(), getNullLogger(t), clt, testNamespace), "'%s' namespace not found", testNamespace)
	})
	t.Run("namespace does not exist in cluster", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: testOtherNamespace,
					UID:  testUID,
				},
			}).
			Build()

		require.Errorf(t, checkNamespace(context.Background(), getNullLogger(t), clt, testNamespace), "'%s' namespace found", testNamespace)
	})
	t.Run("invalid namespace selected", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: testNamespace,
					UID:  testUID,
				},
			}).
			Build()

		require.Errorf(t, checkNamespace(context.Background(), getNullLogger(t), clt, testOtherNamespace), "'%s' namespace found", testNamespace)
	})
}
