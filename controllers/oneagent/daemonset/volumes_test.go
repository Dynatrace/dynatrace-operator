package daemonset

import (
	"testing"
)

func TestPrepareVolumes(t *testing.T) {
	//t.Run(`has root volume`, func(t *testing.T) {
	//	instance := &v1alpha1.DynaKube{}
	//	fullstackSpec := &v1alpha1.FullStackSpec{}
	//	volumes := prepareVolumes(instance, fullstackSpec)
	//
	//	assert.Contains(t, volumes, getRootVolume())
	//	assert.NotContains(t, volumes, getCertificateVolume(instance))
	//	assert.NotContains(t, volumes, getInstallationVolume(fullstackSpec))
	//})
	//t.Run(`has certificate volume`, func(t *testing.T) {
	//	instance := &v1alpha1.DynaKube{
	//		Spec: v1alpha1.DynaKubeSpec{
	//			TrustedCAs: testName,
	//		},
	//	}
	//	fullstackSpec := &v1alpha1.FullStackSpec{}
	//	volumes := prepareVolumes(instance, fullstackSpec)
	//
	//	assert.Contains(t, volumes, getRootVolume())
	//	assert.Contains(t, volumes, getCertificateVolume(instance))
	//	assert.NotContains(t, volumes, getInstallationVolume(fullstackSpec))
	//})
	//t.Run(`has readonly installation volume`, func(t *testing.T) {
	//	instance := &v1alpha1.DynaKube{}
	//	fullstackSpec := &v1alpha1.FullStackSpec{
	//		ReadOnly: v1alpha1.ReadOnlySpec{
	//			Enabled: true,
	//			InstallationVolume: &v1.VolumeSource{
	//				EmptyDir: &v1.EmptyDirVolumeSource{},
	//			},
	//		},
	//	}
	//	volumes := prepareVolumes(instance, fullstackSpec)
	//
	//	assert.Contains(t, volumes, getRootVolume())
	//	assert.NotContains(t, volumes, getCertificateVolume(instance))
	//	assert.Contains(t, volumes, getInstallationVolume(fullstackSpec))
	//})
	//t.Run(`has all volumes`, func(t *testing.T) {
	//	instance := &v1alpha1.DynaKube{
	//		Spec: v1alpha1.DynaKubeSpec{
	//			TrustedCAs: testName,
	//		},
	//	}
	//	fullstackSpec := &v1alpha1.FullStackSpec{
	//		ReadOnly: v1alpha1.ReadOnlySpec{
	//			Enabled: true,
	//			InstallationVolume: &v1.VolumeSource{
	//				EmptyDir: &v1.EmptyDirVolumeSource{},
	//			},
	//		},
	//	}
	//	volumes := prepareVolumes(instance, fullstackSpec)
	//
	//	assert.Contains(t, volumes, getRootVolume())
	//	assert.Contains(t, volumes, getCertificateVolume(instance))
	//	assert.Contains(t, volumes, getInstallationVolume(fullstackSpec))
	//})
}

func TestPrepareVolumeMounts(t *testing.T) {
	//t.Run(`has root volume mount`, func(t *testing.T) {
	//	instance := &v1alpha1.DynaKube{}
	//	fullstackSpec := &v1alpha1.FullStackSpec{}
	//	volumeMounts := prepareVolumeMounts(instance, fullstackSpec)
	//
	//	assert.Contains(t, volumeMounts, getRootMount())
	//	assert.NotContains(t, volumeMounts, getCertificateMount())
	//	assert.NotContains(t, volumeMounts, getInstallationMount())
	//})
	//t.Run(`has certificate volume mount`, func(t *testing.T) {
	//	instance := &v1alpha1.DynaKube{
	//		Spec: v1alpha1.DynaKubeSpec{
	//			TrustedCAs: testName,
	//		},
	//	}
	//	fullstackSpec := &v1alpha1.FullStackSpec{}
	//	volumeMounts := prepareVolumeMounts(instance, fullstackSpec)
	//
	//	assert.Contains(t, volumeMounts, getRootMount())
	//	assert.Contains(t, volumeMounts, getCertificateMount())
	//	assert.NotContains(t, volumeMounts, getInstallationMount())
	//})
	//t.Run(`has installation volume mount`, func(t *testing.T) {
	//	instance := &v1alpha1.DynaKube{}
	//	fullstackSpec := &v1alpha1.FullStackSpec{
	//		ReadOnly: v1alpha1.ReadOnlySpec{
	//			Enabled: true,
	//			InstallationVolume: &v1.VolumeSource{
	//				EmptyDir: &v1.EmptyDirVolumeSource{},
	//			},
	//		},
	//	}
	//	volumeMounts := prepareVolumeMounts(instance, fullstackSpec)
	//	rootMount := getRootMount()
	//	rootMount.ReadOnly = true
	//
	//	assert.Contains(t, volumeMounts, rootMount)
	//	assert.NotContains(t, volumeMounts, getCertificateMount())
	//	assert.Contains(t, volumeMounts, getInstallationMount())
	//})
	//t.Run(`has all volume mount`, func(t *testing.T) {
	//	instance := &v1alpha1.DynaKube{
	//		Spec: v1alpha1.DynaKubeSpec{
	//			TrustedCAs: testName,
	//		},
	//	}
	//	fullstackSpec := &v1alpha1.FullStackSpec{
	//		ReadOnly: v1alpha1.ReadOnlySpec{
	//			Enabled: true,
	//			InstallationVolume: &v1.VolumeSource{
	//				EmptyDir: &v1.EmptyDirVolumeSource{},
	//			},
	//		},
	//	}
	//	volumeMounts := prepareVolumeMounts(instance, fullstackSpec)
	//	rootMount := getRootMount()
	//	rootMount.ReadOnly = true
	//
	//	assert.Contains(t, volumeMounts, rootMount)
	//	assert.Contains(t, volumeMounts, getCertificateMount())
	//	assert.Contains(t, volumeMounts, getInstallationMount())
	//})
}
