package modifiers

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/authtoken"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAuthTokenGetVolumes(t *testing.T) {
	dynakube := dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakubeName,
			Namespace: testNamespaceName,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				Capabilities: []dynatracev1beta1.CapabilityDisplayName{
					dynatracev1beta1.KubeMonCapability.DisplayName,
				},
			},
		},
	}
	t.Run("create volumes", func(t *testing.T) {
		mod := NewAuthTokenModifier(dynakube).(AuthTokenModifier)

		volumes := mod.getVolumes()

		require.NotEmpty(t, volumes)
		assert.Len(t, volumes, 1)
		assert.Equal(t, consts.AuthTokenSecretVolumeName, volumes[0].Name)
		assert.Equal(t, dynakube.ActiveGateAuthTokenSecret(), volumes[0].VolumeSource.Secret.SecretName)
	})
}

func TestAuthTokenGetVolumeMounts(t *testing.T) {
	dynakube := dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakubeName,
			Namespace: testNamespaceName,
			Annotations: map[string]string{
				dynatracev1beta1.AnnotationFeatureActiveGateAuthToken: "true",
			},
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				Capabilities: []dynatracev1beta1.CapabilityDisplayName{
					dynatracev1beta1.KubeMonCapability.DisplayName,
				},
			},
		},
	}
	t.Run("create volumes", func(t *testing.T) {
		mod := NewAuthTokenModifier(dynakube).(AuthTokenModifier)

		volumeMounts := mod.getVolumeMounts()

		require.NotEmpty(t, volumeMounts)
		assert.Len(t, volumeMounts, 1)
		assert.True(t, volumeMounts[0].ReadOnly)
		assert.Equal(t, consts.AuthTokenSecretVolumeName, volumeMounts[0].Name)
		assert.Equal(t, consts.AuthTokenMountPoint, volumeMounts[0].MountPath)
		assert.Equal(t, authtoken.ActiveGateAuthTokenName, volumeMounts[0].SubPath)

	})
}

func TestAuthTokenEnabled(t *testing.T) {

	t.Run("true", func(t *testing.T) {
		dynakube := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakubeName,
				Namespace: testNamespaceName,
				Annotations: map[string]string{
					dynatracev1beta1.AnnotationFeatureActiveGateAuthToken: "true",
				},
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					Capabilities: []dynatracev1beta1.CapabilityDisplayName{
						dynatracev1beta1.KubeMonCapability.DisplayName,
					},
				},
			},
		}
		mod := NewAuthTokenModifier(dynakube).(AuthTokenModifier)

		assert.True(t, mod.Enabled())
	})

	t.Run("false", func(t *testing.T) {
		dynakube := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakubeName,
				Namespace: testNamespaceName,
				Annotations: map[string]string{
					dynatracev1beta1.AnnotationFeatureActiveGateAuthToken: "false",
				},
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					Capabilities: []dynatracev1beta1.CapabilityDisplayName{
						dynatracev1beta1.KubeMonCapability.DisplayName,
					},
				},
			},
		}
		mod := NewAuthTokenModifier(dynakube).(AuthTokenModifier)

		assert.False(t, mod.Enabled())
	})
}

func TestAuthTokenModify(t *testing.T) {
	dynakube := dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakubeName,
			Namespace: testNamespaceName,
			Annotations: map[string]string{
				dynatracev1beta1.AnnotationFeatureActiveGateAuthToken: "true",
			},
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				Capabilities: []dynatracev1beta1.CapabilityDisplayName{
					dynatracev1beta1.KubeMonCapability.DisplayName,
				},
			},
		},
	}
	t.Run("create volumes", func(t *testing.T) {
		mod := NewAuthTokenModifier(dynakube).(AuthTokenModifier)
		builder := createBuilderForTesting()
		builder.AddModifier(mod)

		sts := builder.Build()

		require.NotEmpty(t, sts)
		for _, volume := range mod.getVolumes() {
			stsVolume, err := kubeobjects.GetVolumeByName(sts.Spec.Template.Spec.Volumes, volume.Name)
			require.NotNil(t, stsVolume)
			require.NoError(t, err)
			assert.Equal(t, volume, *stsVolume)
		}

		for _, mounts := range mod.getVolumeMounts() {
			containerMount, err := kubeobjects.GetVolumeMountByName(sts.Spec.Template.Spec.Containers[0].VolumeMounts, mounts.Name)
			require.NotNil(t, containerMount)
			require.NoError(t, err)
			assert.Equal(t, mounts, *containerMount)
		}
	})
}
