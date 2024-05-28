package daemonset

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
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
		instance := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: dynakube.OneAgentSpec{
					HostMonitoring: &dynakube.HostInjectSpec{},
				},
			},
		}
		volumes := prepareVolumes(instance)

		assert.Contains(t, volumes, getRootVolume())
		assert.NotContains(t, volumes, getCertificateVolume(instance))
	})
	t.Run(`has tenant secret volume`, func(t *testing.T) {
		instance := &dynakube.DynaKube{
			ObjectMeta: corev1.ObjectMeta{
				Name: testName,
			},
		}
		volumes := prepareVolumes(instance)

		assert.Contains(t, volumes, getOneAgentSecretVolume(instance))
	})
	t.Run(`has certificate volume`, func(t *testing.T) {
		instance := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				TrustedCAs: testName,
				OneAgent: dynakube.OneAgentSpec{
					HostMonitoring: &dynakube.HostInjectSpec{},
				},
			},
		}
		volumes := prepareVolumes(instance)

		assert.Contains(t, volumes, getRootVolume())
		assert.Contains(t, volumes, getCertificateVolume(instance))
	})
	t.Run(`has http_proxy volume`, func(t *testing.T) {
		instance := &dynakube.DynaKube{}
		instance.Spec =
			dynakube.DynaKubeSpec{
				Proxy: &dynakube.DynaKubeProxy{ValueFrom: proxy.BuildSecretName(instance.Name)},
			}

		volumes := prepareVolumes(instance)

		assert.Contains(t, volumes, getRootVolume())
		assert.Contains(t, volumes, buildHttpProxyVolume(instance))
	})
	t.Run(`has tls volume`, func(t *testing.T) {
		instance := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				TrustedCAs: testName,
				ActiveGate: dynakube.ActiveGateSpec{
					Capabilities: []dynakube.CapabilityDisplayName{
						dynakube.KubeMonCapability.DisplayName,
					},
					TlsSecretName: "testing",
				},
				OneAgent: dynakube.OneAgentSpec{
					HostMonitoring: &dynakube.HostInjectSpec{},
				},
			},
		}
		volumes := prepareVolumes(instance)
		assert.Contains(t, volumes, getActiveGateCaCertVolume(instance))
	})
	t.Run(`csi volume not supported on classicFullStack`, func(t *testing.T) {
		instance := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: dynakube.OneAgentSpec{
					ClassicFullStack: &dynakube.HostInjectSpec{},
				},
			},
		}
		volumes := prepareVolumes(instance)
		assert.NotContains(t, volumes, getCSIStorageVolume(instance))
	})
	t.Run(`has all volumes`, func(t *testing.T) {
		instance := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				TrustedCAs: testName,
				OneAgent: dynakube.OneAgentSpec{
					HostMonitoring: &dynakube.HostInjectSpec{},
				},
				ActiveGate: dynakube.ActiveGateSpec{
					Capabilities: []dynakube.CapabilityDisplayName{
						dynakube.KubeMonCapability.DisplayName,
					},
					TlsSecretName: "testing",
				},
			},
		}
		dsBuilder := hostMonitoring{
			builder{
				dk:             instance,
				hostInjectSpec: instance.Spec.OneAgent.HostMonitoring,
				clusterID:      "",
			},
		}
		ds, err := dsBuilder.BuildDaemonSet()
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
		instance := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: dynakube.OneAgentSpec{
					HostMonitoring: &dynakube.HostInjectSpec{},
				},
			},
		}
		volumeMounts := prepareVolumeMounts(instance)

		assert.Contains(t, volumeMounts, getReadOnlyRootMount())
		assert.NotContains(t, volumeMounts, getActiveGateCaCertVolumeMount())
	})
	t.Run(`has cluster certificate volume mount`, func(t *testing.T) {
		instance := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: dynakube.OneAgentSpec{
					HostMonitoring: &dynakube.HostInjectSpec{},
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
		instance := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: dynakube.OneAgentSpec{
					HostMonitoring: &dynakube.HostInjectSpec{},
				},
				TrustedCAs: testName,
				ActiveGate: dynakube.ActiveGateSpec{
					Capabilities: []dynakube.CapabilityDisplayName{
						dynakube.KubeMonCapability.DisplayName,
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
		instance := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: dynakube.OneAgentSpec{
					ClassicFullStack: &dynakube.HostInjectSpec{},
				},
			},
		}
		volumeMounts := prepareVolumeMounts(instance)

		assert.Contains(t, volumeMounts, getRootMount())
		assert.NotContains(t, volumeMounts, getCSIStorageMount())
	})
	t.Run(`has all volume mounts`, func(t *testing.T) {
		instance := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				TrustedCAs: testName,
				OneAgent: dynakube.OneAgentSpec{
					HostMonitoring: &dynakube.HostInjectSpec{},
				},
				ActiveGate: dynakube.ActiveGateSpec{
					Capabilities: []dynakube.CapabilityDisplayName{
						dynakube.KubeMonCapability.DisplayName,
					},
					TlsSecretName: "testing",
				},
			},
		}
		dsBuilder := hostMonitoring{
			builder{
				dk:             instance,
				hostInjectSpec: instance.Spec.OneAgent.HostMonitoring,
				clusterID:      "",
			},
		}

		podSpec, _ := dsBuilder.podSpec()
		volumeMounts := podSpec.Containers[0].VolumeMounts

		assert.Contains(t, volumeMounts, getReadOnlyRootMount())
		assert.Contains(t, volumeMounts, getActiveGateCaCertVolumeMount())
		assert.Contains(t, volumeMounts, getClusterCaCertVolumeMount())
		assert.Contains(t, volumeMounts, getCSIStorageMount())
	})
	t.Run(`has no volume if proxy is set and proxy ignore feature-flags is used`, func(t *testing.T) {
		instance := &dynakube.DynaKube{
			ObjectMeta: corev1.ObjectMeta{
				Name:      "Dynakube",
				Namespace: "dynatrace",
				Annotations: map[string]string{
					dynakube.AnnotationFeatureOneAgentIgnoreProxy: "true",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				Proxy: &dynakube.DynaKubeProxy{ValueFrom: proxy.BuildSecretName("Dynakube")},
				OneAgent: dynakube.OneAgentSpec{
					HostMonitoring: &dynakube.HostInjectSpec{},
				},
			},
		}

		volumes := prepareVolumes(instance)
		mounts := prepareVolumeMounts(instance)

		assert.NotContains(t, volumes, buildHttpProxyVolume(instance))
		assert.NotContains(t, mounts, getHttpProxyMount())
	})
}
