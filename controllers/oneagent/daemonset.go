package oneagent

import (
	"fmt"
	"strconv"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/utils"
	"github.com/Dynatrace/dynatrace-operator/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/version"
	corev1 "k8s.io/api/core/v1"
)

const (
	hostRootMount = "host-root"

	oneagentInstallationMountName = "oneagent-installation"
	oneagentInstallationMountPath = "/mnt/volume_storage_mount"
	oneagentReadOnlyMode          = "ONEAGENT_READ_ONLY_MODE"
)

type reservedEnvVar struct {
	Name    string
	Default func(ev *corev1.EnvVar)
	Value   *corev1.EnvVar
}

func prepareArgs(instance *dynatracev1alpha1.DynaKube, fs *dynatracev1alpha1.FullStackSpec, feature string, clusterID string) []string {
	args := fs.Args
	if instance.Spec.Proxy != nil && (instance.Spec.Proxy.ValueFrom != "" || instance.Spec.Proxy.Value != "") {
		args = append(args, "--set-proxy=$(https_proxy)")
	}

	if instance.Spec.NetworkZone != "" {
		args = append(args, fmt.Sprintf("--set-network-zone=%s", instance.Spec.NetworkZone))
	}

	if feature == InframonFeature {
		args = append(args, "--set-host-id-source=k8s-node-name")
	} else {
		args = append(args, "--set-host-id-source=auto")
	}

	args = append(args, "--set-host-property=OperatorVersion="+version.Version)

	metadata := deploymentmetadata.NewDeploymentMetadata(clusterID)
	args = append(args, metadata.AsArgs()...)
	return args
}

func prepareEnvVars(instance *dynatracev1alpha1.DynaKube, fs *dynatracev1alpha1.FullStackSpec, feature string, clusterID string) []corev1.EnvVar {
	reserved := getReservedEnvVars(instance, fs, clusterID, feature)
	reservedMap := envVarsToMap(reserved)

	// Split defined environment variables between those reserved and the rest

	var remaining []corev1.EnvVar
	for i := range fs.Env {
		if p := reservedMap[fs.Env[i].Name]; p != nil {
			p.Value = &fs.Env[i]
		} else {
			remaining = append(remaining, fs.Env[i])
		}
	}

	// Add reserved environment variables in that order, and generate a default if unset.
	env := generateDefaultValues(reserved)

	return append(env, remaining...)
}

func generateDefaultValues(reserved []reservedEnvVar) []corev1.EnvVar {
	var env []corev1.EnvVar
	for i := range reserved {
		ev := reserved[i].Value
		if ev == nil {
			ev = &corev1.EnvVar{Name: reserved[i].Name}
			reserved[i].Default(ev)
		}
		env = append(env, *ev)
	}
	return env
}

func envVarsToMap(reserved []reservedEnvVar) map[string]*reservedEnvVar {
	reservedMap := map[string]*reservedEnvVar{}
	for i := range reserved {
		reservedMap[reserved[i].Name] = &reserved[i]
	}
	return reservedMap
}

func getReservedEnvVars(instance *dynatracev1alpha1.DynaKube, fs *dynatracev1alpha1.FullStackSpec, clusterID string, feature string) []reservedEnvVar {
	reserved := getClusterEnvVars(clusterID)

	if feature == InframonFeature {
		reserved = append(reserved, getInfraMonitoringEnvVar())
	}

	if !instance.Status.OneAgent.UseImmutableImage {
		reserved = append(reserved, getImmutableImageEnvVars(instance)...)

		if p := instance.Spec.Proxy; p != nil && (p.Value != "" || p.ValueFrom != "") {
			reserved = append(reserved, getProxyEnvVar(instance, p))
		}
	}

	if fs.ReadOnly.Enabled {
		reserved = append(reserved, getReadOnlyEnvVar())
	}
	return reserved
}

func getReadOnlyEnvVar() reservedEnvVar {
	return reservedEnvVar{
		Name: oneagentReadOnlyMode,
		Default: func(ev *corev1.EnvVar) {
			ev.Value = "true"
		},
	}
}

func getProxyEnvVar(instance *dynatracev1alpha1.DynaKube, proxy *dynatracev1alpha1.DynaKubeProxy) reservedEnvVar {
	return reservedEnvVar{
		Name: "https_proxy",
		Default: func(ev *corev1.EnvVar) {
			if proxy.ValueFrom != "" {
				ev.ValueFrom = &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: instance.Spec.Proxy.ValueFrom},
						Key:                  "proxy",
					},
				}
			} else {
				proxy.Value = instance.Spec.Proxy.Value
			}
		},
	}
}

func getImmutableImageEnvVars(instance *dynatracev1alpha1.DynaKube) []reservedEnvVar {
	return []reservedEnvVar{{
		Name: "ONEAGENT_INSTALLER_DOWNLOAD_TOKEN",
		Default: func(ev *corev1.EnvVar) {
			ev.ValueFrom = &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: instance.Tokens()},
					Key:                  utils.DynatracePaasToken,
				},
			}
		}},
		{
			Name: "ONEAGENT_INSTALLER_SCRIPT_URL",
			Default: func(ev *corev1.EnvVar) {
				ev.Value = fmt.Sprintf("%s/v1/deployment/installer/agent/unix/default/latest?arch=x86&flavor=default", instance.Spec.APIURL)
			},
		},
		{
			Name: "ONEAGENT_INSTALLER_SKIP_CERT_CHECK",
			Default: func(ev *corev1.EnvVar) {
				ev.Value = strconv.FormatBool(instance.Spec.SkipCertCheck)
			},
		},
	}
}

func getInfraMonitoringEnvVar() reservedEnvVar {
	return reservedEnvVar{
		Name: "ONEAGENT_DISABLE_CONTAINER_INJECTION",
		Default: func(ev *corev1.EnvVar) {
			ev.Value = "true"
		},
	}
}

func getClusterEnvVars(clusterID string) []reservedEnvVar {
	return []reservedEnvVar{
		{
			Name: "DT_K8S_NODE_NAME",
			Default: func(ev *corev1.EnvVar) {
				ev.ValueFrom = &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"}}
			},
		},
		{
			Name: "DT_K8S_CLUSTER_ID",
			Default: func(ev *corev1.EnvVar) {
				ev.Value = clusterID
			},
		},
	}
}

func prepareVolumeMounts(instance *dynatracev1alpha1.DynaKube, fs *dynatracev1alpha1.FullStackSpec) []corev1.VolumeMount {
	rootMount := getRootMount()
	var volumeMounts []corev1.VolumeMount

	if instance.Spec.TrustedCAs != "" {
		volumeMounts = append(volumeMounts, getCertificateMount())
	}

	if fs.ReadOnly.Enabled {
		volumeMounts = append(volumeMounts, getInstallationMount())
		rootMount.ReadOnly = true
	}

	volumeMounts = append(volumeMounts, rootMount)
	return volumeMounts
}

func getInstallationMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      oneagentInstallationMountName,
		MountPath: oneagentInstallationMountPath,
	}
}

func getCertificateMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      "certs",
		MountPath: "/mnt/dynatrace/certs",
	}
}

func getRootMount() corev1.VolumeMount {

	return corev1.VolumeMount{
		Name:      hostRootMount,
		MountPath: "/mnt/root",
	}
}

func prepareVolumes(instance *dynatracev1alpha1.DynaKube, fs *dynatracev1alpha1.FullStackSpec) []corev1.Volume {
	volumes := []corev1.Volume{getRootVolume()}

	if instance.Spec.TrustedCAs != "" {
		volumes = append(volumes, getCertificateVolume(instance))
	}

	if fs.ReadOnly.Enabled {
		volumes = append(volumes, getInstallationVolume(fs))
	}

	return volumes
}

func getInstallationVolume(fs *dynatracev1alpha1.FullStackSpec) corev1.Volume {
	return corev1.Volume{
		Name:         oneagentInstallationMountName,
		VolumeSource: fs.ReadOnly.InstallationVolume,
	}
}

func getCertificateVolume(instance *dynatracev1alpha1.DynaKube) corev1.Volume {
	return corev1.Volume{
		Name: "certs",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: instance.Spec.TrustedCAs,
				},
				Items: []corev1.KeyToPath{
					{
						Key:  "certs",
						Path: "certs.pem",
					},
				},
			},
		},
	}
}

func getRootVolume() corev1.Volume {
	return corev1.Volume{
		Name: "host-root",
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: "/",
			},
		},
	}
}
