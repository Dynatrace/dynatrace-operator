package secret

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const testSourceSecretName = "source-secret"

var (
	testLog = logd.Get().WithName("secret-test")
)

func TestReplicate(t *testing.T) {
	ctx := context.Background()

	sourceNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dk-ns",
		},
	}

	targetNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "user-ns",
		},
	}

	data := map[string][]byte{
		"data": []byte("beep"),
	}

	t.Run("create", func(t *testing.T) {
		source := clientSecret(testSourceSecretName, sourceNs.Name, data)
		source.Labels = map[string]string{
			"key": "value",
		}
		clt := fake.NewClientWithIndex(
			targetNs,
			source,
			sourceNs,
		)

		err := Replicate(ctx, Query(clt, clt, testLog), testSourceSecretName, consts.BootstrapperInitSecretName, sourceNs.Name, targetNs.Name)
		require.NoError(t, err)

		var replicated corev1.Secret
		err = clt.Get(context.Background(), client.ObjectKey{Name: consts.BootstrapperInitSecretName, Namespace: targetNs.Name}, &replicated)
		require.NoError(t, err)
		assert.Equal(t, source.Data, replicated.Data)
		assert.Equal(t, source.Labels, replicated.Labels)
	})

	t.Run("already exists => no update + no error", func(t *testing.T) {
		source := clientSecret(testSourceSecretName, sourceNs.Name, data)
		alreadyPresentConfig := clientSecret(consts.BootstrapperInitSecretName, targetNs.Name, nil)
		clt := fake.NewClientWithIndex(
			targetNs,
			source,
			alreadyPresentConfig,
		)

		err := Replicate(ctx, Query(clt, clt, testLog), testSourceSecretName, consts.BootstrapperInitSecretName, sourceNs.Name, targetNs.Name)
		require.NoError(t, err)

		var replicated corev1.Secret
		err = clt.Get(context.Background(), client.ObjectKey{Name: consts.BootstrapperInitSecretName, Namespace: targetNs.Name}, &replicated)
		require.NoError(t, err)
		assert.NotEqual(t, source.Data, replicated.Data)
	})
}

func clientSecret(secretName string, namespaceName string, data map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "core/v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespaceName,
		},
		Data: data,
	}
}
