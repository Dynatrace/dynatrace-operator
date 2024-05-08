package daemonset

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPrepareVolumes(t *testing.T) {
	t.Run("has defaults if instance is nil", func(t *testing.T) {
		volumes := prepareVolumes(nil)

		assert.Contains(t, volumes, getRootVolume())
	})
	t.Run(`has root volume`, func(t *testing.T) {
		instance := &dynatracev1beta2.DynaKube{
			Spec: dynatracev1beta2.DynaKubeSpec{
				OneAgent: dynatracev1beta2.OneAgentSpec{
					HostMonitoring: &dynatracev1beta2.HostInjectSpec{},
				},
			},
		}
		volumes := prepareVolumes(instance)

		assert.Contains(t, volumes, getRootVolume())
		assert.NotContains(t, volumes, getCertificateVolume(instance))
	})
	t.Run(`has tenant secret volume`, func(t *testing.T) {
		instance := &dynatracev1beta2.DynaKube{
			ObjectMeta: corev1.ObjectMeta{
				Name: testName,
			},
		}
		volumes := prepareVolumes(instance)

		assert.Contains(t, volumes, getOneAgentSecretVolume(instance))
	})
	t.Run(`has certificate volume`, func(t *testing.T) {
		instance := &dynatracev1beta2.DynaKube{
			Spec: dynatracev1beta2.DynaKubeSpec{
				TrustedCAs: testName,
				OneAgent: dynatracev1beta2.OneAgentSpec{
					HostMonitoring: &dynatracev1beta2.HostInjectSpec{},
				},
			},
		}
		volumes := prepareVolumes(instance)

		assert.Contains(t, volumes, getRootVolume())
		assert.Contains(t, volumes, getCertificateVolume(instance))
	})
	t.Run(`has http_proxy volume`, func(t *testing.T) {
		instance := &dynatracev1beta2.DynaKube{}
		instance.Spec =
			dynatracev1beta2.DynaKubeSpec{
				Proxy: &dynatracev1beta2.DynaKubeProxy{ValueFrom: proxy.BuildSecretName(instance.Name)},
			}

		volumes := prepareVolumes(instance)

		assert.Contains(t, volumes, getRootVolume())
		assert.Contains(t, volumes, buildHttpProxyVolume(instance))
	})
	t.Run(`has tls volume`, func(t *testing.T) {
		instance := &dynatracev1beta2.DynaKube{
			Spec: dynatracev1beta2.DynaKubeSpec{
				TrustedCAs: testName,
				ActiveGate: dynatracev1beta2.ActiveGateSpec{
					Capabilities: []dynatracev1beta2.CapabilityDisplayName{
						dynatracev1beta2.KubeMonCapability.DisplayName,
					},
					TlsSecretName: "testing",
				},
				OneAgent: dynatracev1beta2.OneAgentSpec{
					HostMonitoring: &dynatracev1beta2.HostInjectSpec{},
				},
			},
		}
		volumes := prepareVolumes(instance)
		assert.Contains(t, volumes, getActiveGateCaCertVolume(instance))
	})
	t.Run(`csi volume not supported on classicFullStack`, func(t *testing.T) {
		instance := &dynatracev1beta2.DynaKube{
			Spec: dynatracev1beta2.DynaKubeSpec{
				OneAgent: dynatracev1beta2.OneAgentSpec{
					ClassicFullStack: &dynatracev1beta2.HostInjectSpec{},
				},
			},
		}
		volumes := prepareVolumes(instance)
		assert.NotContains(t, volumes, getCSIStorageVolume(instance))
	})
	t.Run(`has all volumes`, func(t *testing.T) {
		instance := &dynatracev1beta2.DynaKube{
			Spec: dynatracev1beta2.DynaKubeSpec{
				TrustedCAs: testName,
				OneAgent: dynatracev1beta2.OneAgentSpec{
					HostMonitoring: &dynatracev1beta2.HostInjectSpec{},
				},
				ActiveGate: dynatracev1beta2.ActiveGateSpec{
					Capabilities: []dynatracev1beta2.CapabilityDisplayName{
						dynatracev1beta2.KubeMonCapability.DisplayName,
					},
					TlsSecretName: "testing",
				},
			},
		}
		dsInfo := HostMonitoring{
			builderInfo{
				dynakube:       instance,
				hostInjectSpec: instance.Spec.OneAgent.HostMonitoring,
				clusterID:      "",
			},
		}
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		volumes := ds.Spec.Template.Spec.Volumes

		assert.Contains(t, volumes, getRootVolume())
		assert.Contains(t, volumes, getCertificateVolume(instance))
		assert.Contains(t, volumes, getActiveGateCaCertVolume(instance))
		assert.Contains(t, volumes, getCSIStorageVolume(instance))
	})
}

func TestPrepareVolumeMounts(t *testing.T) {
	t.Run("has defaults if instance is nil", func(t *testing.T) {
		volumeMounts := prepareVolumeMounts(nil)

		assert.Contains(t, volumeMounts, getRootMount())
	})
	t.Run(`has root volume mount`, func(t *testing.T) {
		instance := &dynatracev1beta2.DynaKube{
			Spec: dynatracev1beta2.DynaKubeSpec{
				OneAgent: dynatracev1beta2.OneAgentSpec{
					HostMonitoring: &dynatracev1beta2.HostInjectSpec{},
				},
			},
		}
		volumeMounts := prepareVolumeMounts(instance)

		assert.Contains(t, volumeMounts, getReadOnlyRootMount())
		assert.NotContains(t, volumeMounts, getActiveGateCaCertVolumeMount())
	})
	t.Run(`has cluster certificate volume mount`, func(t *testing.T) {
		instance := &dynatracev1beta2.DynaKube{
			Spec: dynatracev1beta2.DynaKubeSpec{
				OneAgent: dynatracev1beta2.OneAgentSpec{
					HostMonitoring: &dynatracev1beta2.HostInjectSpec{},
				},
				TrustedCAs: testName,
			},
		}
		volumeMounts := prepareVolumeMounts(instance)

		assert.Contains(t, volumeMounts, getReadOnlyRootMount())
		assert.Contains(t, volumeMounts, getClusterCaCertVolumeMount())
		assert.NotContains(t, volumeMounts, getActiveGateCaCertVolumeMount())
	})
	t.Run(`has ActiveGate CA volume mount`, func(t *testing.T) {
		instance := &dynatracev1beta2.DynaKube{
			Spec: dynatracev1beta2.DynaKubeSpec{
				OneAgent: dynatracev1beta2.OneAgentSpec{
					HostMonitoring: &dynatracev1beta2.HostInjectSpec{},
				},
				TrustedCAs: testName,
				ActiveGate: dynatracev1beta2.ActiveGateSpec{
					Capabilities: []dynatracev1beta2.CapabilityDisplayName{
						dynatracev1beta2.KubeMonCapability.DisplayName,
					},
					TlsSecretName: "testing",
				},
			},
		}

		volumeMounts := prepareVolumeMounts(instance)

		assert.Contains(t, volumeMounts, getReadOnlyRootMount())
		assert.Contains(t, volumeMounts, getActiveGateCaCertVolumeMount())
	})
	t.Run(`readonly volume not supported on classicFullStack`, func(t *testing.T) {
		instance := &dynatracev1beta2.DynaKube{
			Spec: dynatracev1beta2.DynaKubeSpec{
				OneAgent: dynatracev1beta2.OneAgentSpec{
					ClassicFullStack: &dynatracev1beta2.HostInjectSpec{},
				},
			},
		}
		volumeMounts := prepareVolumeMounts(instance)

		assert.Contains(t, volumeMounts, getRootMount())
		assert.NotContains(t, volumeMounts, getCSIStorageMount())
	})
	t.Run(`has all volume mounts`, func(t *testing.T) {
		instance := &dynatracev1beta2.DynaKube{
			Spec: dynatracev1beta2.DynaKubeSpec{
				TrustedCAs: testName,
				OneAgent: dynatracev1beta2.OneAgentSpec{
					HostMonitoring: &dynatracev1beta2.HostInjectSpec{},
				},
				ActiveGate: dynatracev1beta2.ActiveGateSpec{
					Capabilities: []dynatracev1beta2.CapabilityDisplayName{
						dynatracev1beta2.KubeMonCapability.DisplayName,
					},
					TlsSecretName: "testing",
				},
			},
		}
		dsInfo := HostMonitoring{
			builderInfo{
				dynakube:       instance,
				hostInjectSpec: instance.Spec.OneAgent.HostMonitoring,
				clusterID:      "",
			},
		}

		podSpec, _ := dsInfo.podSpec()
		volumeMounts := podSpec.Containers[0].VolumeMounts

		assert.Contains(t, volumeMounts, getReadOnlyRootMount())
		assert.Contains(t, volumeMounts, getActiveGateCaCertVolumeMount())
		assert.Contains(t, volumeMounts, getClusterCaCertVolumeMount())
		assert.Contains(t, volumeMounts, getCSIStorageMount())
	})
	t.Run(`has no volume if proxy is set and proxy ignore feature-flags is used`, func(t *testing.T) {
		instance := &dynatracev1beta2.DynaKube{
			ObjectMeta: corev1.ObjectMeta{
				Name:      "Dynakube",
				Namespace: "dynatrace",
				Annotations: map[string]string{
					dynatracev1beta1.AnnotationFeatureOneAgentIgnoreProxy: "true",
				},
			},
			Spec: dynatracev1beta2.DynaKubeSpec{
				Proxy: &dynatracev1beta2.DynaKubeProxy{ValueFrom: proxy.BuildSecretName("Dynakube")},
				OneAgent: dynatracev1beta2.OneAgentSpec{
					HostMonitoring: &dynatracev1beta2.HostInjectSpec{},
				},
			},
		}

		volumes := prepareVolumes(instance)
		mounts := prepareVolumeMounts(instance)

		assert.NotContains(t, volumes, buildHttpProxyVolume(instance))
		assert.NotContains(t, mounts, getHttpProxyMount())
	})
}
