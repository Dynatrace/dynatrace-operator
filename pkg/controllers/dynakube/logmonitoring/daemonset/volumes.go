package daemonset

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/logmonitoring/configsecret"
	corev1 "k8s.io/api/core/v1"
)

const (
	// for configuring the logmonitoring
	configVolumeName      = "config"
	configVolumeMountPath = "/var/lib/dynatrace/oneagent/agent/config/deployment.conf"

	// for the logmonitoring to read/write
	dtLibVolumeName         = "dynatrace-lib"
	dtLibVolumeMountPath    = "/var/lib/dynatrace"
	dtLibVolumePathTemplate = "/tmp/dynatrace-logmonitoring-%s"
	dtLogVolumeName         = "dynatrace-logs"
	dtLogVolumeMountPath    = "/var/log/dynatrace"

	// for the logs that the logmonitoring will ingest
	podLogsVolumeName       = "var-log-pods"
	podLogsVolumePath       = "/var/log/pods"
	dockerLogsVolumeName    = "docker-container-logs"
	dockerLogsVolumePath    = "/var/lib/docker/containers"
	containerLogsVolumeName = "container-logs"
	containerLogsVolumePath = "/var/log/containers"
)

// getConfigVolumeMount provides the VolumeMount for the deployment.conf
func getConfigVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      configVolumeName,
		MountPath: configVolumeMountPath,
		SubPath:   configsecret.DeploymentConfigFilename,
	}
}

// getConfigVolumeMount provides the Volume for the deployment.conf
func getConfigVolume(dkName string) corev1.Volume {
	return corev1.Volume{
		Name: configVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: configsecret.GetSecretName(dkName),
			},
		},
	}
}

// getDTVolumeMounts provides the VolumeMounts for the dynatrace specific folders
func getDTVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      dtLibVolumeName,
			MountPath: dtLibVolumeMountPath,
		},
		{
			Name:      dtLogVolumeName,
			MountPath: dtLogVolumeMountPath,
		},
	}
}

// getDTVolumes provides the Volumes for the dynatrace specific folders
func getDTVolumes(tenantUUID string) []corev1.Volume {
	return []corev1.Volume{
		{
			Name: dtLibVolumeName,
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: fmt.Sprintf(dtLibVolumePathTemplate, tenantUUID),
				},
			},
		},
		{
			Name:         dtLogVolumeName,
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
		},
	}
}

// getIngestVolumeMounts provides the VolumeMounts for the log folders that will be ingested by the logmonitoring
func getIngestVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      dockerLogsVolumeName,
			MountPath: dockerLogsVolumePath,
			ReadOnly:  true,
		},
		{
			Name:      podLogsVolumeName,
			MountPath: podLogsVolumePath,
			ReadOnly:  true,
		},
		{
			Name:      containerLogsVolumeName,
			MountPath: containerLogsVolumePath,
			ReadOnly:  true,
		},
	}
}

// getIngestVolumeMounts provides the VolumeMounts for the log folders that will be ingested by the logmonitoring
func getIngestVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: dockerLogsVolumeName,
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: dockerLogsVolumePath,
				},
			},
		},
		{
			Name: podLogsVolumeName,
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: podLogsVolumePath,
				},
			},
		},
		{
			Name: containerLogsVolumeName,
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: containerLogsVolumePath,
				},
			},
		},
	}
}

func getVolumeMounts() []corev1.VolumeMount {
	mounts := []corev1.VolumeMount{}
	mounts = append(mounts, getConfigVolumeMount())
	mounts = append(mounts, getDTVolumeMounts()...)
	mounts = append(mounts, getIngestVolumeMounts()...)

	return mounts
}

func getVolumes(dkName, tenantUUID string) []corev1.Volume {
	volumes := []corev1.Volume{}
	volumes = append(volumes, getConfigVolume(dkName))
	volumes = append(volumes, getDTVolumes(tenantUUID)...)
	volumes = append(volumes, getIngestVolumes()...)

	return volumes
}
