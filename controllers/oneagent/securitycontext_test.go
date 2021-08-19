package oneagent

const (
	testKey   = "test-key"
	testValue = "test-value"

	testReadOnlyInstallationVolumePath = "/my/path/to/a/directory"
)

//
//func TestNewPodSpecForCR_ReadOnlyMode(t *testing.T) {
//	instance := &dynatracev1alpha1.DynaKube{
//		Spec: dynatracev1alpha1.DynaKubeSpec{
//			InfraMonitoring: dynatracev1alpha1.InfraMonitoringSpec{
//				FullStackSpec: dynatracev1alpha1.FullStackSpec{
//					Enabled: true,
//				},
//				ReadOnly:      dynatracev1alpha1.ReadOnlySpec{
//					Enabled: true,
//					InstallationVolume: &corev1.VolumeSource{
//						HostPath: &corev1.HostPathVolumeSource{
//							Path: testReadOnlyInstallationVolumePath,
//						},
//					},
//				},
//			},
//		},
//	}
//	podSpec := newPodSpecForCR(instance, &instance.Spec.InfraMonitoring.FullStackSpec, "", true, logger.NewDTLogger(), "")
//
//	assert.NotNil(t, podSpec)
//	assert.Contains(t, podSpec.Volumes, corev1.Volume{
//		Name: oneagentInstallationMountName,
//		VolumeSource: corev1.VolumeSource{
//			HostPath: &corev1.HostPathVolumeSource{
//				Path: testReadOnlyInstallationVolumePath,
//				Type: nil,
//			},
//		},
//	})
//	assert.Contains(t, podSpec.Containers[0].Env, corev1.EnvVar{
//		Name:  oneagentReadOnlyMode,
//		Value: "true",
//	})
//	assert.Contains(t, podSpec.Containers[0].Env, corev1.EnvVar{
//		Name:  enableVolumeStorage,
//		Value: "true",
//	})
//
//	oneagentInstallationMountExists := false
//	for _, volumeMount := range podSpec.Containers[0].VolumeMounts {
//		if volumeMount.Name == hostRootMount {
//			assert.True(t, volumeMount.ReadOnly)
//		}
//		if volumeMount.Name == oneagentInstallationMountName {
//			assert.Equal(t, oneagentInstallationMountPath, volumeMount.MountPath)
//			oneagentInstallationMountExists = true
//		}
//	}
//
//	assert.True(t, oneagentInstallationMountExists)
//}
