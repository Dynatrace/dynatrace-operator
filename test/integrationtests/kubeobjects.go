// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package integrationtests

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateNamespace(t *testing.T, clt client.Client, namespace string) {
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	err := clt.Create(t.Context(), &ns)
	require.NoError(t, err)
}

func CreateKubernetesObject(t *testing.T, clt client.Client, object client.Object) {
	err := clt.Create(t.Context(), object)
	require.NoError(t, err)
}

func CreateDynakube(t *testing.T, clt client.Client, dk *dynakube.DynaKube) {
	dkStatus := dk.Status

	err := clt.Create(t.Context(), dk)
	require.NoError(t, err)

	dk.Status = dkStatus

	err = dk.UpdateStatus(t.Context(), clt)
	require.NoError(t, err)
}
