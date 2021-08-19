package daemonset

import (
	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func TestPrepareVolumes(t *testing.T) {
	t.Run(`has root volume`, func(t *testing.T) {
		instance := &v1alpha1.DynaKube{}
		readOnlySpec := &v1alpha1.ReadOnlySpec{}
		volumes := prepareVolumes(instance)

		assert.Contains(t, volumes, getRootVolume())
		assert.NotContains(t, volumes, getCertificateVolume(instance))
		assert.NotContains(t, volumes, getInstallationVolume(readOnlySpec))
	})
	t.Run(`has certificate volume`, func(t *testing.T) {
		instance := &v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				TrustedCAs: testName,
			},
		}
		readOnlySpec := &v1alpha1.ReadOnlySpec{}
		volumes := prepareVolumes(instance)

		assert.Contains(t, volumes, getRootVolume())
		assert.Contains(t, volumes, getCertificateVolume(instance))
		assert.NotContains(t, volumes, getInstallationVolume(readOnlySpec))
	})
	t.Run(`has readonly installation volume`, func(t *testing.T) {
		readOnlySpec := &v1alpha1.ReadOnlySpec{
			Enabled: true,
		}
		instance := &v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				InfraMonitoring: v1alpha1.InfraMonitoringSpec{
					ReadOnly: *readOnlySpec,
				},
			},
		}
		dsInfo := InfraMonitoring{
			builderInfo{
				instance:      instance,
				fullstackSpec: &v1alpha1.FullStackSpec{Enabled: true},
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
		assert.Contains(t, volumes, getInstallationVolume(readOnlySpec))
	})
	t.Run(`has all volumes`, func(t *testing.T) {
		readOnlySpec := &v1alpha1.ReadOnlySpec{
			Enabled: true,
		}
		instance := &v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				TrustedCAs: testName,
				InfraMonitoring: v1alpha1.InfraMonitoringSpec{
					ReadOnly: *readOnlySpec,
				},
			},
		}
		dsInfo := InfraMonitoring{
			builderInfo{
				instance:      instance,
				fullstackSpec: &v1alpha1.FullStackSpec{Enabled: true},
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
		assert.Contains(t, volumes, getInstallationVolume(readOnlySpec))
	})
}

func TestPrepareVolumeMounts(t *testing.T) {
	t.Run(`has root volume mount`, func(t *testing.T) {
		instance := &v1alpha1.DynaKube{}
		volumeMounts := prepareVolumeMounts(instance)

		assert.Contains(t, volumeMounts, getRootMount())
		assert.NotContains(t, volumeMounts, getCertificateMount())
		assert.NotContains(t, volumeMounts, getInstallationMount())
	})
	t.Run(`has certificate volume mount`, func(t *testing.T) {
		instance := &v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				TrustedCAs: testName,
			},
		}
		volumeMounts := prepareVolumeMounts(instance)

		assert.Contains(t, volumeMounts, getRootMount())
		assert.Contains(t, volumeMounts, getCertificateMount())
		assert.NotContains(t, volumeMounts, getInstallationMount())
	})
	t.Run(`has installation volume mount`, func(t *testing.T) {
		instance := &v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				InfraMonitoring: v1alpha1.InfraMonitoringSpec{
					FullStackSpec: v1alpha1.FullStackSpec{
						Enabled: true,
					},
					ReadOnly: v1alpha1.ReadOnlySpec{
						Enabled: true,
						InstallationVolume: &v1.VolumeSource{
							EmptyDir: &v1.EmptyDirVolumeSource{},
						},
					},
				},
			},
		}
		dsInfo := InfraMonitoring{
			builderInfo{
				instance:      instance,
				fullstackSpec: &instance.Spec.InfraMonitoring.FullStackSpec,
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
		instance := &v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				TrustedCAs: testName,
				InfraMonitoring: v1alpha1.InfraMonitoringSpec{
					FullStackSpec: v1alpha1.FullStackSpec{
						Enabled: true,
					},
					ReadOnly: v1alpha1.ReadOnlySpec{
						Enabled: true,
						InstallationVolume: &v1.VolumeSource{
							EmptyDir: &v1.EmptyDirVolumeSource{},
						},
					},
				},
			},
		}
		dsInfo := InfraMonitoring{
			builderInfo{
				instance:      instance,
				fullstackSpec: &instance.Spec.InfraMonitoring.FullStackSpec,
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
