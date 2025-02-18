package secret

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestSecretCreation(t *testing.T) {
	ctx := context.Background()

	t.Run("creates secret if it does not exist", func(t *testing.T) {
		clt := fake.NewFakeClient()

		dk := createDynaKube()

		err := ensureOpenSignalAPISecret(ctx, clt, clt, &dk)
		require.NoError(t, err)

		var secret corev1.Secret
		err = clt.Get(ctx, types.NamespacedName{Name: telemetryApiCredentialsSecretName, Namespace: dk.Namespace}, &secret)
		require.NoError(t, err)
		assert.NotEmpty(t, secret)
		require.NotNil(t, meta.FindStatusCondition(*dk.Conditions(), secretConditionType))
		assert.Equal(t, conditions.SecretCreatedReason, meta.FindStatusCondition(*dk.Conditions(), secretConditionType).Reason)
	})

}

func createDynaKube() dynakube.DynaKube {
	dk := dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-dk",
		},
		Spec: dynakube.DynaKubeSpec{},
	}

	return dk
}
