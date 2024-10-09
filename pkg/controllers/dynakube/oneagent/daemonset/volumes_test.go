package daemonset

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPrepareVolumes(t *testing.T) {
	t.Run("has defaults if dk is nil", func(t *testing.T) {
		volumes := prepareVolumes(nil)

		assert.Contains(t, volumes, getRootVolume())
	})
	t.Run(`has root volume`, func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: dynakube.OneAgentSpec{
					HostMonitoring: &dynakube.HostInjectSpec{},
				},
			},
		}
		volumes := prepareVolumes(dk)

		assert.Contains(t, volumes, getRootVolume())
		assert.NotContains(t, volumes, getCertificateVolume(dk))
	})
	t.Run(`has tenant secret volume`, func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: corev1.ObjectMeta{
				Name: testName,
			},
		}
		volumes := prepareVolumes(dk)

		assert.Contains(t, volumes, getOneAgentSecretVolume(dk))
	})
	t.Run(`has certificate volume`, func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				TrustedCAs: testName,
				OneAgent: dynakube.OneAgentSpec{
					HostMonitoring: &dynakube.HostInjectSpec{},
				},
			},
		}
		volumes := prepareVolumes(dk)

		assert.Contains(t, volumes, getRootVolume())
		assert.Contains(t, volumes, getCertificateVolume(dk))
	})
	t.Run(`has http_proxy volume`, func(t *testing.T) {
		dk := &dynakube.DynaKube{}
		dk.Spec =
			dynakube.DynaKubeSpec{
				Proxy: &value.Source{ValueFrom: proxy.BuildSecretName(dk.Name)},
			}

		volumes := prepareVolumes(dk)

		assert.Contains(t, volumes, getRootVolume())
		assert.Contains(t, volumes, buildHttpProxyVolume(dk))
	})
	t.Run(`has tls volume`, func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				TrustedCAs: testName,
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.KubeMonCapability.DisplayName,
					},
					TlsSecretName: "testing",
				},
				OneAgent: dynakube.OneAgentSpec{
					HostMonitoring: &dynakube.HostInjectSpec{},
				},
			},
		}
		volumes := prepareVolumes(dk)
		assert.Contains(t, volumes, getActiveGateCaCertVolume(dk))
	})
	t.Run(`csi volume not supported on classicFullStack`, func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: dynakube.OneAgentSpec{
					ClassicFullStack: &dynakube.HostInjectSpec{},
				},
			},
		}
		volumes := prepareVolumes(dk)
		assert.NotContains(t, volumes, getCSIStorageVolume(dk))
	})
	t.Run(`has all volumes`, func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				TrustedCAs: testName,
				OneAgent: dynakube.OneAgentSpec{
					HostMonitoring: &dynakube.HostInjectSpec{},
				},
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.KubeMonCapability.DisplayName,
					},
					TlsSecretName: "testing",
				},
			},
		}
		dsBuilder := hostMonitoring{
			builder{
				dk:             dk,
				hostInjectSpec: dk.Spec.OneAgent.HostMonitoring,
				clusterID:      "",
			},
		}
		ds, err := dsBuilder.BuildDaemonSet()
		require.NoError(t, err)

		volumes := ds.Spec.Template.Spec.Volumes

		assert.Contains(t, volumes, getRootVolume())
		assert.Contains(t, volumes, getCertificateVolume(dk))
		assert.Contains(t, volumes, getActiveGateCaCertVolume(dk))
		assert.Contains(t, volumes, getCSIStorageVolume(dk))
	})
}

func TestPrepareVolumeMounts(t *testing.T) {
	t.Run("has defaults if dk is nil", func(t *testing.T) {
		volumeMounts := prepareVolumeMounts(nil)

		assert.Contains(t, volumeMounts, getRootMount())
	})
	t.Run(`has root volume mount`, func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: dynakube.OneAgentSpec{
					HostMonitoring: &dynakube.HostInjectSpec{},
				},
			},
		}
		volumeMounts := prepareVolumeMounts(dk)

		assert.Contains(t, volumeMounts, getReadOnlyRootMount())
		assert.NotContains(t, volumeMounts, getActiveGateCaCertVolumeMount())
	})
	t.Run(`has cluster certificate volume mount`, func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: dynakube.OneAgentSpec{
					HostMonitoring: &dynakube.HostInjectSpec{},
				},
				TrustedCAs: testName,
			},
		}
		volumeMounts := prepareVolumeMounts(dk)

		assert.Contains(t, volumeMounts, getReadOnlyRootMount())
		assert.Contains(t, volumeMounts, getClusterCaCertVolumeMount())
		assert.NotContains(t, volumeMounts, getActiveGateCaCertVolumeMount())
	})
	t.Run(`has ActiveGate CA volume mount`, func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: dynakube.OneAgentSpec{
					HostMonitoring: &dynakube.HostInjectSpec{},
				},
				TrustedCAs: testName,
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.KubeMonCapability.DisplayName,
					},
					TlsSecretName: "testing",
				},
			},
		}

		volumeMounts := prepareVolumeMounts(dk)

		assert.Contains(t, volumeMounts, getReadOnlyRootMount())
		assert.Contains(t, volumeMounts, getActiveGateCaCertVolumeMount())
	})
	t.Run(`readonly volume not supported on classicFullStack`, func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: dynakube.OneAgentSpec{
					ClassicFullStack: &dynakube.HostInjectSpec{},
				},
			},
		}
		volumeMounts := prepareVolumeMounts(dk)

		assert.Contains(t, volumeMounts, getRootMount())
		assert.NotContains(t, volumeMounts, getCSIStorageMount())
	})
	t.Run(`has all volume mounts`, func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				TrustedCAs: testName,
				OneAgent: dynakube.OneAgentSpec{
					HostMonitoring: &dynakube.HostInjectSpec{},
				},
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.KubeMonCapability.DisplayName,
					},
					TlsSecretName: "testing",
				},
			},
		}
		dsBuilder := hostMonitoring{
			builder{
				dk:             dk,
				hostInjectSpec: dk.Spec.OneAgent.HostMonitoring,
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
		dk := &dynakube.DynaKube{
			ObjectMeta: corev1.ObjectMeta{
				Name:      "Dynakube",
				Namespace: "dynatrace",
				Annotations: map[string]string{
					dynakube.AnnotationFeatureOneAgentIgnoreProxy: "true", //nolint:staticcheck
				},
			},
			Spec: dynakube.DynaKubeSpec{
				Proxy: &value.Source{ValueFrom: proxy.BuildSecretName("Dynakube")},
				OneAgent: dynakube.OneAgentSpec{
					HostMonitoring: &dynakube.HostInjectSpec{},
				},
			},
		}

		volumes := prepareVolumes(dk)
		mounts := prepareVolumeMounts(dk)

		assert.NotContains(t, volumes, buildHttpProxyVolume(dk))
		assert.NotContains(t, mounts, getHttpProxyMount())
	})
}
