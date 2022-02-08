package daemonset

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPrepareVolumes(t *testing.T) {
	t.Run(`has root volume`, func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{}
		volumes := prepareVolumes(instance)

		assert.Contains(t, volumes, getRootVolume())
		assert.NotContains(t, volumes, getCertificateVolume(instance))
	})
	t.Run(`has certificate volume`, func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				TrustedCAs: testName,
			},
		}
		volumes := prepareVolumes(instance)

		assert.Contains(t, volumes, getRootVolume())
		assert.Contains(t, volumes, getCertificateVolume(instance))
	})
	t.Run(`has tls volume`, func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				TrustedCAs: testName,
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					Capabilities: []dynatracev1beta1.CapabilityDisplayName{
						dynatracev1beta1.KubeMonCapability.DisplayName,
					},
					TlsSecretName: "testing",
				},
			},
		}
		volumes := prepareVolumes(instance)
		assert.Contains(t, volumes, getTLSVolume(instance))
	})
	t.Run(`has csi volume`, func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: corev1.ObjectMeta{
				Annotations: map[string]string{
					dynatracev1beta1.AnnotationFeatureReadOnlyOneAgent: "true",
				},
			},
		}
		volumes := prepareVolumes(instance)
		assert.Contains(t, volumes, getCSIStorageVolume())
	})
	t.Run(`has all volumes`, func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: corev1.ObjectMeta{
				Annotations: map[string]string{
					dynatracev1beta1.AnnotationFeatureReadOnlyOneAgent: "true",
				},
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				TrustedCAs: testName,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					HostMonitoring: &dynatracev1beta1.HostMonitoringSpec{},
				},
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					Capabilities: []dynatracev1beta1.CapabilityDisplayName{
						dynatracev1beta1.KubeMonCapability.DisplayName,
					},
					TlsSecretName: "testing",
				},
			},
		}
		dsInfo := HostMonitoring{
			builderInfo{
				instance:       instance,
				hostInjectSpec: &instance.Spec.OneAgent.HostMonitoring.HostInjectSpec,
				clusterId:      "",
			},
			HostMonitoringFeature,
		}
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		volumes := ds.Spec.Template.Spec.Volumes

		assert.Contains(t, volumes, getRootVolume())
		assert.Contains(t, volumes, getCertificateVolume(instance))
		assert.Contains(t, volumes, getTLSVolume(instance))
		assert.Contains(t, volumes, getCSIStorageVolume())
	})
}

func TestPrepareVolumeMounts(t *testing.T) {
	t.Run(`has root volume mount`, func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{}
		volumeMounts := prepareVolumeMounts(instance)

		assert.Contains(t, volumeMounts, getRootMount())
		assert.NotContains(t, volumeMounts, getCertificateMount())
	})
	t.Run(`has certificate volume mount`, func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				TrustedCAs: testName,
			},
		}
		volumeMounts := prepareVolumeMounts(instance)

		assert.Contains(t, volumeMounts, getRootMount())
		assert.Contains(t, volumeMounts, getCertificateMount())
	})
	t.Run(`has tls volume mount`, func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				TrustedCAs: testName,
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					Capabilities: []dynatracev1beta1.CapabilityDisplayName{
						dynatracev1beta1.KubeMonCapability.DisplayName,
					},
					TlsSecretName: "testing",
				},
			},
		}

		volumeMounts := prepareVolumeMounts(instance)

		assert.Contains(t, volumeMounts, getRootMount())
		assert.Contains(t, volumeMounts, getTLSMount())
	})
	t.Run(`has readonly volume mounts`, func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: corev1.ObjectMeta{
				Annotations: map[string]string{
					dynatracev1beta1.AnnotationFeatureReadOnlyOneAgent: "true",
				},
			},
		}

		volumeMounts := prepareVolumeMounts(instance)

		assert.Contains(t, volumeMounts, getReadOnlyRootMount())
		assert.Contains(t, volumeMounts, getCSIStorageMount())
	})
	t.Run(`has all volume mounts`, func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: corev1.ObjectMeta{
				Annotations: map[string]string{
					dynatracev1beta1.AnnotationFeatureReadOnlyOneAgent: "true",
				},
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				TrustedCAs: testName,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					HostMonitoring: &dynatracev1beta1.HostMonitoringSpec{},
				},
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					Capabilities: []dynatracev1beta1.CapabilityDisplayName{
						dynatracev1beta1.KubeMonCapability.DisplayName,
					},
					TlsSecretName: "testing",
				},
			},
		}
		dsInfo := HostMonitoring{
			builderInfo{
				instance:       instance,
				hostInjectSpec: &instance.Spec.OneAgent.HostMonitoring.HostInjectSpec,
				clusterId:      "",
			},
			HostMonitoringFeature,
		}

		volumeMounts := dsInfo.podSpec().Containers[0].VolumeMounts

		assert.Contains(t, volumeMounts, getReadOnlyRootMount())
		assert.Contains(t, volumeMounts, getCertificateMount())
		assert.Contains(t, volumeMounts, getTLSMount())
		assert.Contains(t, volumeMounts, getCSIStorageMount())
	})
}
