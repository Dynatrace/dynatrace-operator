package daemonset

import (
	"testing"

	dynatracev1 "github.com/Dynatrace/dynatrace-operator/api/v1"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestPrepareVolumes(t *testing.T) {
	t.Run(`has root volume`, func(t *testing.T) {
		instance := &dynatracev1.DynaKube{}
		volumes := prepareVolumes(instance)

		assert.Contains(t, volumes, getRootVolume())
		assert.NotContains(t, volumes, getCertificateVolume(instance))
		assert.NotContains(t, volumes, getInstallationVolume(instance))
	})
	t.Run(`has certificate volume`, func(t *testing.T) {
		instance := &dynatracev1.DynaKube{
			Spec: dynatracev1.DynaKubeSpec{
				TrustedCAs: testName,
			},
		}
		volumes := prepareVolumes(instance)

		assert.Contains(t, volumes, getRootVolume())
		assert.Contains(t, volumes, getCertificateVolume(instance))
		assert.NotContains(t, volumes, getInstallationVolume(instance))
	})
	t.Run(`has readonly installation volume`, func(t *testing.T) {
		instance := &dynatracev1.DynaKube{
			Spec: dynatracev1.DynaKubeSpec{
				OneAgent: dynatracev1.OneAgentSpec{
					HostMonitoring: &dynatracev1.HostMonitoringSpec{},
				},
			},
		}
		dsInfo := InfraMonitoring{
			builderInfo{
				instance:      instance,
				hostInjectSpec: &instance.Spec.OneAgent.HostMonitoring.HostInjectSpec,
				logger:        logger.NewDTLogger(),
				clusterId:     "",
				relatedImage:  "",
			},
		}
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		volumes := ds.Spec.Template.Spec.Volumes

		assert.Contains(t, volumes, getRootVolume())
		assert.NotContains(t, volumes, getCertificateVolume(instance))
		assert.Contains(t, volumes, getInstallationVolume(instance))
	})
	t.Run(`has all volumes`, func(t *testing.T) {
		instance := &dynatracev1.DynaKube{
			Spec: dynatracev1.DynaKubeSpec{
				TrustedCAs: testName,
				OneAgent: dynatracev1.OneAgentSpec{
					HostMonitoring: &dynatracev1.HostMonitoringSpec{},
				},
			},
		}
		dsInfo := InfraMonitoring{
			builderInfo{
				instance:      instance,
				hostInjectSpec: &instance.Spec.OneAgent.HostMonitoring.HostInjectSpec,
				logger:        logger.NewDTLogger(),
				clusterId:     "",
				relatedImage:  "",
			},
		}
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		volumes := ds.Spec.Template.Spec.Volumes

		assert.Contains(t, volumes, getRootVolume())
		assert.Contains(t, volumes, getCertificateVolume(instance))
		assert.Contains(t, volumes, getInstallationVolume(instance))
	})
}

func TestPrepareVolumeMounts(t *testing.T) {
	t.Run(`has root volume mount`, func(t *testing.T) {
		instance := &dynatracev1.DynaKube{}
		volumeMounts := prepareVolumeMounts(instance)

		assert.Contains(t, volumeMounts, getRootMount())
		assert.NotContains(t, volumeMounts, getCertificateMount())
		assert.NotContains(t, volumeMounts, getInstallationMount())
	})
	t.Run(`has certificate volume mount`, func(t *testing.T) {
		instance := &dynatracev1.DynaKube{
			Spec: dynatracev1.DynaKubeSpec{
				TrustedCAs: testName,
			},
		}
		volumeMounts := prepareVolumeMounts(instance)

		assert.Contains(t, volumeMounts, getRootMount())
		assert.Contains(t, volumeMounts, getCertificateMount())
		assert.NotContains(t, volumeMounts, getInstallationMount())
	})
	t.Run(`has installation volume mount`, func(t *testing.T) {
		instance := &dynatracev1.DynaKube{
			Spec: dynatracev1.DynaKubeSpec{
				OneAgent: dynatracev1.OneAgentSpec{
					HostMonitoring: &dynatracev1.HostMonitoringSpec{
						InstallationVolume: &corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				},
			},
		}
		dsInfo := InfraMonitoring{
			builderInfo{
				instance:      instance,
				hostInjectSpec: &instance.Spec.OneAgent.HostMonitoring.HostInjectSpec,
				logger:        logger.NewDTLogger(),
				clusterId:     "",
				relatedImage:  "",
			},
		}
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		volumeMounts := ds.Spec.Template.Spec.Containers[0].VolumeMounts
		rootMount := getRootMount()
		rootMount.ReadOnly = true

		assert.Contains(t, volumeMounts, rootMount)
		assert.NotContains(t, volumeMounts, getCertificateMount())
		assert.Contains(t, volumeMounts, getInstallationMount())
	})
	t.Run(`has all volume mount`, func(t *testing.T) {
		instance := &dynatracev1.DynaKube{
			Spec: dynatracev1.DynaKubeSpec{
				TrustedCAs: testName,
				OneAgent: dynatracev1.OneAgentSpec{
					HostMonitoring: &dynatracev1.HostMonitoringSpec{
						InstallationVolume: &corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				},
			},
		}
		dsInfo := InfraMonitoring{
			builderInfo{
				instance:      instance,
				hostInjectSpec: &instance.Spec.OneAgent.HostMonitoring.HostInjectSpec,
				logger:        logger.NewDTLogger(),
				clusterId:     "",
				relatedImage:  "",
			},
		}
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		volumeMounts := ds.Spec.Template.Spec.Containers[0].VolumeMounts

		rootMount := getRootMount()
		rootMount.ReadOnly = true

		assert.Contains(t, volumeMounts, rootMount)
		assert.Contains(t, volumeMounts, getCertificateMount())
		assert.Contains(t, volumeMounts, getInstallationMount())
	})
}
