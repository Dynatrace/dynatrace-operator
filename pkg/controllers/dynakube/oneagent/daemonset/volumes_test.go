package daemonset

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/proxy"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
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
				OneAgent: oneagent.Spec{
					HostMonitoring: &oneagent.HostInjectSpec{},
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
				OneAgent: oneagent.Spec{
					HostMonitoring: &oneagent.HostInjectSpec{},
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
				OneAgent: oneagent.Spec{
					HostMonitoring: &oneagent.HostInjectSpec{},
				},
			},
		}
		volumes := prepareVolumes(dk)
		assert.Contains(t, volumes, getActiveGateCaCertVolume(dk))
	})
	t.Run(`has automatically created AG tls volume`, func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: corev1.ObjectMeta{
				Name: "dynakube",
			},
			Spec: dynakube.DynaKubeSpec{
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.KubeMonCapability.DisplayName,
					},
				},
				TrustedCAs: testName,
				OneAgent: oneagent.Spec{
					HostMonitoring: &oneagent.HostInjectSpec{},
				},
			},
		}
		volumes := prepareVolumes(dk)
		assert.Contains(t, volumes, getActiveGateCaCertVolume(dk))
	})
	t.Run(`csi volume not supported on classicFullStack`, func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					ClassicFullStack: &oneagent.HostInjectSpec{},
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
				OneAgent: oneagent.Spec{
					HostMonitoring: &oneagent.HostInjectSpec{},
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
				OneAgent: oneagent.Spec{
					HostMonitoring: &oneagent.HostInjectSpec{},
				},
			},
		}
		volumeMounts := prepareVolumeMounts(dk)

		assert.Contains(t, volumeMounts, getReadOnlyRootMount())
		assert.NotContains(t, volumeMounts, getClusterCaCertVolumeMount())
		assert.NotContains(t, volumeMounts, getActiveGateCaCertVolumeMount())
	})
	t.Run(`has cluster certificate volume mount`, func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					HostMonitoring: &oneagent.HostInjectSpec{},
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
				OneAgent: oneagent.Spec{
					HostMonitoring: &oneagent.HostInjectSpec{},
				},
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
		assert.NotContains(t, volumeMounts, getClusterCaCertVolumeMount())
		assert.Contains(t, volumeMounts, getActiveGateCaCertVolumeMount())
	})
	t.Run(`has automatically created ActiveGate CA volume mount`, func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: corev1.ObjectMeta{
				Name: "dynakube",
			},
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					HostMonitoring: &oneagent.HostInjectSpec{},
				},
				TrustedCAs: testName,
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.KubeMonCapability.DisplayName,
					},
				},
			},
		}

		volumeMounts := prepareVolumeMounts(dk)

		assert.Contains(t, volumeMounts, getReadOnlyRootMount())
		assert.Contains(t, volumeMounts, getClusterCaCertVolumeMount())
		assert.Contains(t, volumeMounts, getActiveGateCaCertVolumeMount())
	})
	t.Run(`readonly volume not supported on classicFullStack`, func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					ClassicFullStack: &oneagent.HostInjectSpec{},
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
				OneAgent: oneagent.Spec{
					HostMonitoring: &oneagent.HostInjectSpec{},
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
					exp.OAProxyIgnoredKey: "true", //nolint:staticcheck
				},
			},
			Spec: dynakube.DynaKubeSpec{
				Proxy: &value.Source{ValueFrom: proxy.BuildSecretName("Dynakube")},
				OneAgent: oneagent.Spec{
					HostMonitoring: &oneagent.HostInjectSpec{},
				},
			},
		}

		volumes := prepareVolumes(dk)
		mounts := prepareVolumeMounts(dk)

		assert.NotContains(t, volumes, buildHttpProxyVolume(dk))
		assert.NotContains(t, mounts, getHttpProxyMount())
	})
}

func TestVolumesAndVolumeMountsVsCSIDriver(t *testing.T) {
	dkHostMonitoring := &dynakube.DynaKube{
		Spec: dynakube.DynaKubeSpec{
			OneAgent: oneagent.Spec{
				HostMonitoring: &oneagent.HostInjectSpec{},
			},
		},
	}
	dkCloudNativeFullStack := &dynakube.DynaKube{
		Spec: dynakube.DynaKubeSpec{
			OneAgent: oneagent.Spec{
				CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
			},
		},
	}
	dkClassicFullStack := &dynakube.DynaKube{
		Spec: dynakube.DynaKubeSpec{
			OneAgent: oneagent.Spec{
				ClassicFullStack: &oneagent.HostInjectSpec{},
			},
		},
	}

	type oneAgentVolumeTest struct {
		testName                string
		dk                      *dynakube.DynaKube
		csi                     bool
		csiVolume               bool
		storageVolume           bool
		rootReadOnlyVolumeMount bool
	}

	testCases := []oneAgentVolumeTest{
		{
			testName:                "hostMonitoring w/ CSI driver",
			dk:                      dkHostMonitoring,
			csi:                     true,
			csiVolume:               true,
			storageVolume:           false,
			rootReadOnlyVolumeMount: true,
		},
		{
			testName:                "hostMonitoring w/o CSI driver",
			dk:                      dkHostMonitoring,
			csi:                     false,
			csiVolume:               false,
			storageVolume:           true,
			rootReadOnlyVolumeMount: true,
		},

		{
			testName:                "cloudNativeFullStack w/ CSI driver",
			dk:                      dkCloudNativeFullStack,
			csi:                     true,
			csiVolume:               true,
			storageVolume:           false,
			rootReadOnlyVolumeMount: true,
		},
		{
			testName:                "cloudNativeFullStack w/o CSI driver",
			dk:                      dkCloudNativeFullStack,
			csi:                     false,
			csiVolume:               false,
			storageVolume:           true,
			rootReadOnlyVolumeMount: true,
		},

		{
			testName:                "classicFullStack w/o CSI driver",
			dk:                      dkClassicFullStack,
			csi:                     false,
			csiVolume:               false,
			storageVolume:           false,
			rootReadOnlyVolumeMount: false,
		},
	}

	for _, tc := range testCases {
		t.Run("Volumes:"+tc.testName, func(t *testing.T) {
			testVolumesVsCSIDriver(t, tc.dk, tc.csi, tc.csiVolume, tc.storageVolume)
		})

		t.Run("VolumeMounts:"+tc.testName, func(t *testing.T) {
			testVolumeMountsVsCSIDriver(t, tc.dk, tc.csi, tc.csiVolume, tc.storageVolume, tc.rootReadOnlyVolumeMount)
		})
	}
}

func testVolumesVsCSIDriver(t *testing.T, dk *dynakube.DynaKube, csi bool, csiVolume bool, storageVolume bool) {
	if csi {
		installconfig.SetModulesOverride(t, installconfig.Modules{CSIDriver: true})
	} else {
		installconfig.SetModulesOverride(t, installconfig.Modules{CSIDriver: false})
	}

	volumes := prepareVolumes(dk)

	if csiVolume {
		assert.Contains(t, volumes, getCSIStorageVolume(dk))
	} else {
		assert.NotContains(t, volumes, getCSIStorageVolume(dk))
	}

	if storageVolume {
		assert.Contains(t, volumes, getStorageVolume(dk))
	} else {
		assert.NotContains(t, volumes, getStorageVolume(dk))
	}
}

func testVolumeMountsVsCSIDriver(t *testing.T, dk *dynakube.DynaKube, csi bool, csiVolume bool, storageVolume bool, rootReadOnlyVolumeMount bool) { //nolint:revive
	if csi {
		installconfig.SetModulesOverride(t, installconfig.Modules{CSIDriver: true})
	} else {
		installconfig.SetModulesOverride(t, installconfig.Modules{CSIDriver: false})
	}

	volumeMounts := prepareVolumeMounts(dk)

	if csiVolume {
		assert.Contains(t, volumeMounts, getCSIStorageMount())
	} else {
		assert.NotContains(t, volumeMounts, getCSIStorageMount())
	}

	if storageVolume {
		assert.Contains(t, volumeMounts, getStorageVolumeMount(dk))
	} else {
		assert.NotContains(t, volumeMounts, getStorageVolumeMount(dk))
	}

	if rootReadOnlyVolumeMount {
		assert.Contains(t, volumeMounts, getReadOnlyRootMount())
	} else {
		assert.Contains(t, volumeMounts, getRootMount())
	}
}
