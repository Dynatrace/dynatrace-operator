package status

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubesystem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testUUID = "test-uuid"
)

func TestSetDynakubeStatus(t *testing.T) {
	ctx := context.Background()
	t.Run("set status", func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{}
		clt := fake.NewClient(&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: kubesystem.Namespace,
				UID:  testUUID,
			},
		})
		err := SetKubeSystemUUIDInStatus(ctx, instance, clt)

		require.NoError(t, err)
		assert.Equal(t, testUUID, instance.Status.KubeSystemUUID)
	})
	t.Run("error querying kube system uid", func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{}
		clt := fake.NewClient()

		err := SetKubeSystemUUIDInStatus(ctx, instance, clt)
		require.EqualError(t, err, "namespaces \"kube-system\" not found")
	})

	t.Run("don't query kube system uid if already set", func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{}
		instance.Status.KubeSystemUUID = testUUID
		clt := fake.NewClient()

		err := SetKubeSystemUUIDInStatus(ctx, instance, clt)
		require.NoError(t, err)
	})
}
