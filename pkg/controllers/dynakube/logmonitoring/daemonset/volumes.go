package daemonset

import (
	"fmt"
	"slices"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/logmonitoring/configsecret"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

const (
	// for configuring the logmonitoring
	configVolumeName      = "config"
	configVolumeMountPath = "/var/lib/dynatrace/oneagent/agent/config/deployment.conf"

	// for the logmonitoring configurations to read/write
	dtLibVolumeName                 = "dynatrace-lib"
	dtLibVolumeMountPath            = "/var/lib/dynatrace"
	dtLibVolumeMountSubPathTemplate = "logmonitoring-%s"
	dtLibVolumeHostPath             = oneagent.StorageVolumeDefaultHostPath

	// for the logmonitoring logs to read/write
	dtLogVolumeName      = "dynatrace-logs"
	dtLogVolumeMountPath = "/tmp/dynatrace"

	// for the logs that the logmonitoring will ingest
	dockerLogsVolumeName = "docker-container-logs"
	dockerLogsVolumePath = "/var/lib/docker/containers"
	logsVolumeHostPath   = "/var/log"
	logsVolumeName       = "var-log"
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
func getDTVolumeMounts(tenantUUID string) corev1.VolumeMount {
	return corev1.VolumeMount{

		Name:      dtLibVolumeName,
		SubPath:   fmt.Sprintf(dtLibVolumeMountSubPathTemplate, tenantUUID),
		MountPath: dtLibVolumeMountPath,
	}
}

// getDTVolumeMounts provides the VolumeMounts for the dynatrace specific folders
func getDTLogVolumeMounts(tenantUUID string) corev1.VolumeMount {
	return corev1.VolumeMount{

		Name:      dtLogVolumeName,
		SubPath:   fmt.Sprintf(dtLibVolumeMountSubPathTemplate, tenantUUID),
		MountPath: dtLogVolumeMountPath,
	}
}

// getDTVolumes provides the Volumes for the dynatrace specific folders
func getDTVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: dtLibVolumeName,
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: dtLibVolumeHostPath,
					Type: ptr.To(corev1.HostPathDirectoryOrCreate),
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
			Name:      logsVolumeName,
			MountPath: logsVolumeHostPath,
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
					Type: ptr.To(corev1.HostPathDirectoryOrCreate),
				},
			},
		},
		{
			Name: logsVolumeName,
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: logsVolumeHostPath,
					Type: ptr.To(corev1.HostPathDirectory),
				},
			},
		},
	}
}

func getVolumeMounts(tenantUUID string) []corev1.VolumeMount {
	return slices.Concat(
		[]corev1.VolumeMount{
			getConfigVolumeMount(),
			getDTVolumeMounts(tenantUUID),
			getDTLogVolumeMounts(tenantUUID),
		},
		getIngestVolumeMounts(),
	)
}

func getVolumes(dkName string) []corev1.Volume {
	return slices.Concat(
		[]corev1.Volume{getConfigVolume(dkName)},
		getDTVolumes(),
		getIngestVolumes(),
	)
}
