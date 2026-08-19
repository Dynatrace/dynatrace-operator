// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package statefulset

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/telemetryingest"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/authtoken"
	"github.com/Dynatrace/dynatrace-operator/test/integrationtests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testTenantUUID     = "abc12345"
	testKubeSystemUUID = "12345"
)

func TestStatefulSet(t *testing.T) {
	clt := integrationtests.SetupTestEnvironment(t)

	ctx := t.Context()

	dk := getTestDynakube()
	dk.Status = dynakube.DynaKubeStatus{
		ActiveGate: activegate.Status{
			ConnectionInfo: communication.ConnectionInfo{
				TenantUUID: testTenantUUID,
			},
			VersionStatus: status.VersionStatus{
				ImageID: "thisismytenant.com/linux/activegate@sha256:312a5fafebb134371dc05e3e0ad00641bd44fde2a31b70dca5edbc708f2e76cb",
			},
		},
		KubeSystemUUID: testKubeSystemUUID,
	}
	dk.Spec.TelemetryIngest = &telemetryingest.Spec{}

	integrationtests.CreateNamespace(t, ctx, clt, testNamespaceName)
	integrationtests.CreateDynakube(t, ctx, clt, &dk)
	integrationtests.CreateKubernetesObject(t, ctx, clt, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakubeName + activegate.AuthTokenSecretSuffix,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{authtoken.ActiveGateAuthTokenName: []byte(testToken)},
	})

	mcap := capability.NewMultiCapability(&dk)
	reconciler := NewReconciler(clt, clt)

	err := reconciler.Reconcile(ctx, &dk, mcap)
	require.NoError(t, err)

	dk.Spec.ActiveGate.UseEphemeralVolume = new(true)
	err = reconciler.Reconcile(ctx, &dk, mcap)
	require.NoError(t, err)
}

func TestStatefulSetUserVolumesAndMounts(t *testing.T) {
	clt := integrationtests.SetupTestEnvironment(t)
	ctx := t.Context()

	dk := getTestDynakube()
	dk.Status = dynakube.DynaKubeStatus{
		ActiveGate: activegate.Status{
			ConnectionInfo: communication.ConnectionInfo{
				TenantUUID: testTenantUUID,
			},
			VersionStatus: status.VersionStatus{
				ImageID: "thisismytenant.com/linux/activegate@sha256:312a5fafebb134371dc05e3e0ad00641bd44fde2a31b70dca5edbc708f2e76cb",
			},
		},
		KubeSystemUUID: testKubeSystemUUID,
	}

	userVolume := corev1.Volume{
		Name: "my-user-volume",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	userVolumeMount := corev1.VolumeMount{
		Name:      "my-user-volume",
		MountPath: "/my/custom/path",
	}
	dk.Spec.ActiveGate.Volumes = []corev1.Volume{userVolume}
	dk.Spec.ActiveGate.VolumeMounts = []corev1.VolumeMount{userVolumeMount}

	integrationtests.CreateNamespace(t, ctx, clt, testNamespaceName)
	integrationtests.CreateDynakube(t, ctx, clt, &dk)
	integrationtests.CreateKubernetesObject(t, ctx, clt, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakubeName + activegate.AuthTokenSecretSuffix,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{authtoken.ActiveGateAuthTokenName: []byte(testToken)},
	})

	mcap := capability.NewMultiCapability(&dk)
	reconciler := NewReconciler(clt, clt)
	require.NoError(t, reconciler.Reconcile(ctx, &dk, mcap))

	sts := &appsv1.StatefulSet{}
	require.NoError(t, clt.Get(ctx, client.ObjectKey{Name: capability.BuildServiceName(dk.Name), Namespace: dk.Namespace}, sts))
	assert.Contains(t, sts.Spec.Template.Spec.Volumes, userVolume)
	require.Len(t, sts.Spec.Template.Spec.Containers, 1)
	assert.Contains(t, sts.Spec.Template.Spec.Containers[0].VolumeMounts, userVolumeMount)
}

func TestStatefulSetUserVolumesAndMountsFail(t *testing.T) {
	clt := integrationtests.SetupTestEnvironment(t)
	ctx := t.Context()

	dk := getTestDynakube()
	dk.Status = dynakube.DynaKubeStatus{
		ActiveGate: activegate.Status{
			ConnectionInfo: communication.ConnectionInfo{
				TenantUUID: testTenantUUID,
			},
			VersionStatus: status.VersionStatus{
				ImageID: "thisismytenant.com/linux/activegate@sha256:312a5fafebb134371dc05e3e0ad00641bd44fde2a31b70dca5edbc708f2e76cb",
			},
		},
		KubeSystemUUID: testKubeSystemUUID,
	}

	userVolume := corev1.Volume{
		Name: "my-user-volume-typo",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	userVolumeMount := corev1.VolumeMount{
		Name:      "my-user-volume",
		MountPath: "/my/custom/path",
	}
	dk.Spec.ActiveGate.Volumes = []corev1.Volume{userVolume}
	dk.Spec.ActiveGate.VolumeMounts = []corev1.VolumeMount{userVolumeMount}

	integrationtests.CreateNamespace(t, ctx, clt, testNamespaceName)
	integrationtests.CreateDynakube(t, ctx, clt, &dk)
	integrationtests.CreateKubernetesObject(t, ctx, clt, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakubeName + activegate.AuthTokenSecretSuffix,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{authtoken.ActiveGateAuthTokenName: []byte(testToken)},
	})

	mcap := capability.NewMultiCapability(&dk)
	reconciler := NewReconciler(clt, clt)
	require.Error(t, reconciler.Reconcile(ctx, &dk, mcap))
}
