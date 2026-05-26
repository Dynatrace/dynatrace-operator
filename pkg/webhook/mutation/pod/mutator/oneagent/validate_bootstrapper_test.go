package oneagent

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/volumes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestValidateBootstrapperSecretVolumeMounts_Mutate(t *testing.T) {
	inputMount := corev1.VolumeMount{Name: volumes.InputVolumeName, MountPath: "/dynatrace"}
	otherMount := corev1.VolumeMount{Name: "user-volume", MountPath: "/data"}

	t.Run("empty pod => no error", func(t *testing.T) {
		pod := &corev1.Pod{}

		require.NoError(t, validateBootstrapperSecretVolumeMounts(logd.Get(), pod, noExemption))
	})

	t.Run("only unrelated mounts => no error", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{
					{Name: "init", VolumeMounts: []corev1.VolumeMount{otherMount}},
				},
				Containers: []corev1.Container{
					{Name: "app", VolumeMounts: []corev1.VolumeMount{otherMount}},
				},
			},
		}

		require.NoError(t, validateBootstrapperSecretVolumeMounts(logd.Get(), pod, noExemption))
	})

	t.Run("init container mounts reserved input volume => error", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{
					{Name: "malicious-init", VolumeMounts: []corev1.VolumeMount{inputMount}},
				},
				Containers: []corev1.Container{
					{Name: "app"},
				},
			},
		}

		err := validateBootstrapperSecretVolumeMounts(logd.Get(), pod, noExemption)
		require.Error(t, err)
		assertVolumeMountMutatorError(t, err, "malicious-init")
	})

	t.Run("regular container mounts reserved input volume => error", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "first", VolumeMounts: []corev1.VolumeMount{otherMount}},
					{Name: "malicious", VolumeMounts: []corev1.VolumeMount{inputMount}},
				},
			},
		}

		err := validateBootstrapperSecretVolumeMounts(logd.Get(), pod, noExemption)
		require.Error(t, err)
		assertVolumeMountMutatorError(t, err, "malicious")
	})

	t.Run("user init container named like install container is not skipped => error", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{
					{Name: dtwebhook.InstallContainerName, VolumeMounts: []corev1.VolumeMount{inputMount}},
				},
			},
		}

		err := validateBootstrapperSecretVolumeMounts(logd.Get(), pod, noExemption)
		require.Error(t, err)
		assertVolumeMountMutatorError(t, err, dtwebhook.InstallContainerName)
	})

	t.Run("init container error wins over regular container error", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{
					{Name: "bad-init", VolumeMounts: []corev1.VolumeMount{inputMount}},
				},
				Containers: []corev1.Container{
					{Name: "bad-app", VolumeMounts: []corev1.VolumeMount{inputMount}},
				},
			},
		}

		err := validateBootstrapperSecretVolumeMounts(logd.Get(), pod, noExemption)
		require.Error(t, err)
		assertVolumeMountMutatorError(t, err, "bad-init")
	})
}

func TestValidateBootstrapperSecretVolumeMounts_Reinvoke(t *testing.T) {
	// In Reinvoke the operator-owned install container legitimately mounts the
	// reserved input volume and must be skipped.
	skip := dtwebhook.InstallContainerName

	inputMount := corev1.VolumeMount{Name: volumes.InputVolumeName, MountPath: "/dynatrace"}

	t.Run("install container with input mount is skipped => no error", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{
					{Name: dtwebhook.InstallContainerName, VolumeMounts: []corev1.VolumeMount{inputMount}},
				},
				Containers: []corev1.Container{
					{Name: "app"},
				},
			},
		}

		require.NoError(t, validateBootstrapperSecretVolumeMounts(logd.Get(), pod, skip))
	})

	t.Run("only operator install container present => no error", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{
					{Name: dtwebhook.InstallContainerName, VolumeMounts: []corev1.VolumeMount{inputMount}},
				},
			},
		}

		require.NoError(t, validateBootstrapperSecretVolumeMounts(logd.Get(), pod, skip))
	})

	t.Run("newly-added init container mounts reserved input volume => error", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{
					{Name: dtwebhook.InstallContainerName, VolumeMounts: []corev1.VolumeMount{inputMount}},
					{Name: "newly-added-init", VolumeMounts: []corev1.VolumeMount{inputMount}},
				},
				Containers: []corev1.Container{
					{Name: "app"},
				},
			},
		}

		err := validateBootstrapperSecretVolumeMounts(logd.Get(), pod, skip)
		require.Error(t, err)
		assertVolumeMountMutatorError(t, err, "newly-added-init")
	})

	t.Run("newly-added regular container mounts reserved input volume => error", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{
					{Name: dtwebhook.InstallContainerName, VolumeMounts: []corev1.VolumeMount{inputMount}},
				},
				Containers: []corev1.Container{
					{Name: "app"},
					{Name: "newly-added-sidecar", VolumeMounts: []corev1.VolumeMount{inputMount}},
				},
			},
		}

		err := validateBootstrapperSecretVolumeMounts(logd.Get(), pod, skip)
		require.Error(t, err)
		assertVolumeMountMutatorError(t, err, "newly-added-sidecar")
	})
}

func TestValidateBootstrapperSecretVolumes_Mutate(t *testing.T) {
	t.Run("empty pod => no error", func(t *testing.T) {
		pod := &corev1.Pod{}

		require.NoError(t, validateBootstrapperSecretVolumes(logd.Get(), pod, noExemption))
	})

	t.Run("only unrelated volumes => no error", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{
						Name: "user-secret",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{SecretName: "some-app-secret"},
						},
					},
					{
						Name:         "user-configmap",
						VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{}},
					},
				},
			},
		}

		require.NoError(t, validateBootstrapperSecretVolumes(logd.Get(), pod, noExemption))
	})

	t.Run("volume mounts reserved bootstrapper init secret => error", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{
						Name: "evil-volume",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{SecretName: consts.BootstrapperInitSecretName},
						},
					},
				},
			},
		}

		err := validateBootstrapperSecretVolumes(logd.Get(), pod, noExemption)
		require.Error(t, err)
		assertVolumeMutatorError(t, err, "evil-volume", consts.BootstrapperInitSecretName)
	})

	t.Run("volume mounts reserved bootstrapper certs secret => error", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{
						Name: "evil-certs",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{SecretName: consts.BootstrapperInitCertsSecretName},
						},
					},
				},
			},
		}

		err := validateBootstrapperSecretVolumes(logd.Get(), pod, noExemption)
		require.Error(t, err)
		assertVolumeMutatorError(t, err, "evil-certs", consts.BootstrapperInitCertsSecretName)
	})

	t.Run("user volume named like input volume is not skipped => error", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{
						Name: volumes.InputVolumeName,
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{SecretName: consts.BootstrapperInitSecretName},
						},
					},
				},
			},
		}

		err := validateBootstrapperSecretVolumes(logd.Get(), pod, noExemption)
		require.Error(t, err)
		assertVolumeMutatorError(t, err, volumes.InputVolumeName, consts.BootstrapperInitSecretName)
	})

	t.Run("projected volume with reserved bootstrapper init secret source => error", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{
						Name: "evil-projected",
						VolumeSource: corev1.VolumeSource{
							Projected: &corev1.ProjectedVolumeSource{
								Sources: []corev1.VolumeProjection{
									{Secret: &corev1.SecretProjection{
										LocalObjectReference: corev1.LocalObjectReference{Name: consts.BootstrapperInitSecretName},
									}},
								},
							},
						},
					},
				},
			},
		}

		err := validateBootstrapperSecretVolumes(logd.Get(), pod, noExemption)
		require.Error(t, err)
		assertVolumeMutatorError(t, err, "evil-projected", consts.BootstrapperInitSecretName)
	})

	t.Run("projected volume with reserved bootstrapper certs secret source => error", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{
						Name: "evil-projected-certs",
						VolumeSource: corev1.VolumeSource{
							Projected: &corev1.ProjectedVolumeSource{
								Sources: []corev1.VolumeProjection{
									{Secret: &corev1.SecretProjection{
										LocalObjectReference: corev1.LocalObjectReference{Name: consts.BootstrapperInitCertsSecretName},
									}},
								},
							},
						},
					},
				},
			},
		}

		err := validateBootstrapperSecretVolumes(logd.Get(), pod, noExemption)
		require.Error(t, err)
		assertVolumeMutatorError(t, err, "evil-projected-certs", consts.BootstrapperInitCertsSecretName)
	})

	t.Run("projected volume with unrelated source => no error", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{
						Name: "harmless-projected",
						VolumeSource: corev1.VolumeSource{
							Projected: &corev1.ProjectedVolumeSource{
								Sources: []corev1.VolumeProjection{
									{Secret: &corev1.SecretProjection{
										LocalObjectReference: corev1.LocalObjectReference{Name: "user-secret"},
									}},
									{ConfigMap: &corev1.ConfigMapProjection{}},
								},
							},
						},
					},
				},
			},
		}

		require.NoError(t, validateBootstrapperSecretVolumes(logd.Get(), pod, noExemption))
	})
}

func TestValidateBootstrapperSecretVolumes_Reinvoke(t *testing.T) {
	// In Reinvoke the operator-owned input volume legitimately references the
	// bootstrapper secrets and must be skipped.
	skip := volumes.InputVolumeName

	operatorInputVolume := corev1.Volume{
		Name: volumes.InputVolumeName,
		VolumeSource: corev1.VolumeSource{
			Projected: &corev1.ProjectedVolumeSource{
				Sources: []corev1.VolumeProjection{
					{Secret: &corev1.SecretProjection{
						LocalObjectReference: corev1.LocalObjectReference{Name: consts.BootstrapperInitSecretName},
					}},
					{Secret: &corev1.SecretProjection{
						LocalObjectReference: corev1.LocalObjectReference{Name: consts.BootstrapperInitCertsSecretName},
					}},
				},
			},
		},
	}

	t.Run("operator-owned input volume is skipped => no error", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{operatorInputVolume},
			},
		}

		require.NoError(t, validateBootstrapperSecretVolumes(logd.Get(), pod, skip))
	})

	t.Run("only operator-owned input volume present (direct secret) => no error", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{
						Name: volumes.InputVolumeName,
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{SecretName: consts.BootstrapperInitSecretName},
						},
					},
				},
			},
		}

		require.NoError(t, validateBootstrapperSecretVolumes(logd.Get(), pod, skip))
	})

	t.Run("newly-added volume with bootstrapper init secret => error", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					operatorInputVolume,
					{
						Name: "newly-added-volume",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{SecretName: consts.BootstrapperInitSecretName},
						},
					},
				},
			},
		}

		err := validateBootstrapperSecretVolumes(logd.Get(), pod, skip)
		require.Error(t, err)
		assertVolumeMutatorError(t, err, "newly-added-volume", consts.BootstrapperInitSecretName)
	})

	t.Run("newly-added volume with bootstrapper certs secret => error", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					operatorInputVolume,
					{
						Name: "newly-added-certs-volume",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{SecretName: consts.BootstrapperInitCertsSecretName},
						},
					},
				},
			},
		}

		err := validateBootstrapperSecretVolumes(logd.Get(), pod, skip)
		require.Error(t, err)
		assertVolumeMutatorError(t, err, "newly-added-certs-volume", consts.BootstrapperInitCertsSecretName)
	})

	t.Run("newly-added projected volume sourcing bootstrapper secret => error", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					operatorInputVolume,
					{
						Name: "newly-added-projected",
						VolumeSource: corev1.VolumeSource{
							Projected: &corev1.ProjectedVolumeSource{
								Sources: []corev1.VolumeProjection{
									{Secret: &corev1.SecretProjection{
										LocalObjectReference: corev1.LocalObjectReference{Name: consts.BootstrapperInitCertsSecretName},
									}},
								},
							},
						},
					},
				},
			},
		}

		err := validateBootstrapperSecretVolumes(logd.Get(), pod, skip)
		require.Error(t, err)
		assertVolumeMutatorError(t, err, "newly-added-projected", consts.BootstrapperInitCertsSecretName)
	})
}

func assertVolumeMountMutatorError(t *testing.T, err error, containerName string) {
	t.Helper()

	var mutErr dtwebhook.MutatorError
	require.ErrorAs(t, err, &mutErr, "error must be a MutatorError")

	var vmErr bootstrapperSecretVolumeMountError
	require.ErrorAs(t, err, &vmErr, "error must wrap bootstrapperSecretVolumeMountError")
	assert.Equal(t, containerName, vmErr.ContainerName)

	pod := &corev1.Pod{}
	mutErr.SetAnnotations(pod)
	assert.Equal(t, "false", pod.Annotations[AnnotationInjected])
	assert.Equal(t, BootstrapperSecretMountedReason, pod.Annotations[AnnotationReason])
}

func assertVolumeMutatorError(t *testing.T, err error, volumeName, secretName string) {
	t.Helper()

	var mutErr dtwebhook.MutatorError
	require.ErrorAs(t, err, &mutErr, "error must be a MutatorError")

	var volErr bootstrapperSecretVolumeError
	require.ErrorAs(t, err, &volErr, "error must wrap bootstrapperSecretVolumeError")
	assert.Equal(t, volumeName, volErr.VolumeName)
	assert.Equal(t, secretName, volErr.SecretName)

	pod := &corev1.Pod{}
	mutErr.SetAnnotations(pod)
	assert.Equal(t, "false", pod.Annotations[AnnotationInjected])
	assert.Equal(t, BootstrapperSecretMountedReason, pod.Annotations[AnnotationReason])
}
