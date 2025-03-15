package bootstrapperconfig

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestReplicate(t *testing.T) {
	ctx := context.Background()
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dk",
			Namespace: "dk-ns",
		},
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "user-ns",
		},
	}

	data := map[string][]byte{
		"data": []byte("beep"),
	}

	t.Run("create", func(t *testing.T) {
		source := clientSecret(GetSourceSecretName(dk.Name), dk.Namespace, data)
		source.Labels = map[string]string{
			"key": "value",
		}
		clt := fake.NewClientWithIndex(
			dk,
			ns,
			source,
		)

		err := Replicate(ctx, *dk, secret.Query(clt, clt, log), ns.Name)
		require.NoError(t, err)

		var replicated corev1.Secret
		err = clt.Get(context.Background(), client.ObjectKey{Name: consts.BootstrapperInitSecretName, Namespace: ns.Name}, &replicated)
		require.NoError(t, err)
		assert.Equal(t, source.Data, replicated.Data)
		assert.Equal(t, source.Labels, replicated.Labels)
	})

	t.Run("already exists => no update + no error", func(t *testing.T) {
		source := clientSecret(GetSourceSecretName(dk.Name), dk.Namespace, data)
		alreadyPresent := clientSecret(consts.BootstrapperInitSecretName, ns.Name, nil)
		clt := fake.NewClientWithIndex(
			dk,
			ns,
			source,
			alreadyPresent,
		)

		err := Replicate(ctx, *dk, secret.Query(clt, clt, log), ns.Name)
		require.NoError(t, err)

		var replicated corev1.Secret
		err = clt.Get(context.Background(), client.ObjectKey{Name: consts.BootstrapperInitSecretName, Namespace: ns.Name}, &replicated)
		require.NoError(t, err)
		assert.NotEqual(t, source.Data, replicated.Data)
	})
}
