package integrationtests

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateNamespace(t *testing.T, ctx context.Context, clt client.Client, namespace string) {
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	err := clt.Create(ctx, &ns)
	require.NoError(t, err)
}

func CreateKubernetesObject(t *testing.T, ctx context.Context, clt client.Client, object client.Object) {
	err := clt.Create(ctx, object)
	require.NoError(t, err)
}

func CreateDynakube(t *testing.T, ctx context.Context, clt client.Client, dk *dynakube.DynaKube) {
	dkStatus := dk.Status

	err := clt.Create(ctx, dk)
	require.NoError(t, err)

	dk.Status = dkStatus

	err = dk.UpdateStatus(ctx, clt)
	require.NoError(t, err)
}