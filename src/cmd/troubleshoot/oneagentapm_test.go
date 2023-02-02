package troubleshoot

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestTroubleshootOneAgentAPM(t *testing.T) {
	t.Run("oneagentAPM does not exist in cluster", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: testNamespace,
					UID:  testUID,
				},
			}).
			Build()

		troubleshootCtx := troubleshootContext{context: context.TODO(), apiReader: clt}
		assert.NoError(t, checkOneAgentAPM(clt, &troubleshootCtx))
	})
}
