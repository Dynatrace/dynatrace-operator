package databases

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	probePort = int32(8080)

	// Keep in sync with helm chart
	defaultServiceAccount = "dynatrace-database-extensions-executor"

	executorLabelKey     = "extensions.dynatrace.com/executor.id"
	datasourceLabelKey   = "extensions.dynatrace.com/datasource"
	datasourceLabelValue = "sql"
	userDataVolumeName   = "user-data"
	userDataMountPath    = "/var/userdata"
	tokenVolumeName      = "auth-token"
	tokenMountPath       = "/var/run/dynatrace/executor/token"
	certsVolumeName      = "https-certs"
	certsMountPath       = "/certs"

	conditionType = "DatabaseDatasourcesAvailable"
)

func buildAllLabels(dk *dynakube.DynaKube, dbex extensions.DatabaseSpec) (map[string]string, map[string]string, map[string]string) {
	appLabels := labels.NewAppLabels(labels.DatabaseDatasourceLabel, dk.Name, labels.DatabaseDatasourceLabel, "")

	deploymentLabels := appLabels.BuildLabels()
	// Remove empty version label
	delete(deploymentLabels, labels.AppVersionLabel)

	matchLabels := appLabels.BuildMatchLabels()

	templateLabels := maps.Clone(deploymentLabels)
	templateLabels[executorLabelKey] = dbex.ID
	templateLabels[datasourceLabelKey] = datasourceLabelValue

	if dbex.Labels != nil {
		temp := maps.Clone(dbex.Labels)
		maps.Copy(temp, templateLabels)
		templateLabels = temp
	}

	return deploymentLabels, matchLabels, templateLabels
}

func buildServiceAccountName(dbex extensions.DatabaseSpec) string {
	if dbex.ServiceAccountName != "" {
		return dbex.ServiceAccountName
	}

	return defaultServiceAccount
}

func buildContainer(dk *dynakube.DynaKube, dbex extensions.DatabaseSpec) corev1.Container {
	container := corev1.Container{
		Name:            "database-datasource",
		Image:           dk.Spec.Templates.DatabaseExecutor.ImageRef.String(),
		ImagePullPolicy: corev1.PullIfNotPresent,
		Args:            buildContainerArgs(dk),
		Env:             buildContainerEnvs(),
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/health/live",
					Port: intstr.IntOrString{IntVal: probePort},
				},
			},
			InitialDelaySeconds: 15,
			PeriodSeconds:       10,
			TimeoutSeconds:      2,
			FailureThreshold:    3,
			SuccessThreshold:    1,
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/health/ready",
					Port: intstr.IntOrString{IntVal: probePort},
				},
			},
			InitialDelaySeconds: 10,
			PeriodSeconds:       5,
			TimeoutSeconds:      2,
			FailureThreshold:    3,
			SuccessThreshold:    1,
		},
		Resources:       buildContainerResources(dbex.Resources),
		SecurityContext: buildContainerSecurityContext(),
		VolumeMounts:    buildVolumeMounts(dbex),
	}

	return container
}

func buildContainerArgs(dk *dynakube.DynaKube) []string {
	return []string{
		"--podid=$(POD_NAME)",
		fmt.Sprintf("--url=https://%s:%d", dk.Extensions().GetServiceNameFQDN(), consts.OtelCollectorComPort),
		"--idtoken=" + tokenMountPath + "/" + tokenVolumeName,
	}
}

func buildContainerEnvs() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
	}
}

func buildVolumeMounts(dbex extensions.DatabaseSpec) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      userDataVolumeName,
			MountPath: userDataMountPath,
		},
		{
			Name:      tokenVolumeName,
			MountPath: tokenMountPath,
			ReadOnly:  true,
		},
		{
			Name:      certsVolumeName,
			MountPath: certsMountPath,
			ReadOnly:  true,
		},
	}

	return append(volumeMounts, dbex.VolumeMounts...)
}

func buildVolumes(dk *dynakube.DynaKube, dbex extensions.DatabaseSpec) []corev1.Volume {
	volumes := []corev1.Volume{
		{
			Name: userDataVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: tokenVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: dk.Extensions().GetTokenSecretName(),
					Items: []corev1.KeyToPath{
						{
							Key:  consts.DatasourceTokenSecretKey,
							Path: tokenVolumeName,
						},
					},
				},
			},
		},
		{
			Name: certsVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: dk.Extensions().GetTLSSecretName(),
				},
			},
		},
	}

	return append(volumes, dbex.Volumes...)
}

func buildContainerResources(custom *corev1.ResourceRequirements) corev1.ResourceRequirements {
	if custom != nil {
		return *custom
	}

	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("256Mi"),
			corev1.ResourceCPU:    resource.MustParse("250m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
			corev1.ResourceCPU:    resource.MustParse("500m"),
		},
	}
}

func buildPodSecurityContext() *corev1.PodSecurityContext {
	return &corev1.PodSecurityContext{
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
		RunAsNonRoot: ptr.To(true),
		// TODO: decide on UID/GID
		RunAsGroup: ptr.To(int64(1000)),
		RunAsUser:  ptr.To(int64(1000)),
	}
}

func buildContainerSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		Privileged:               ptr.To(false),
		AllowPrivilegeEscalation: ptr.To(false),
		ReadOnlyRootFilesystem:   ptr.To(true),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				"ALL",
			},
		},
	}
}

func deleteDeployments(ctx context.Context, clt client.Client, dk *dynakube.DynaKube, keep []string) error {
	deployments := &appsv1.DeploymentList{}

	deploymentLabels, _, _ := buildAllLabels(dk, extensions.DatabaseSpec{})
	if err := clt.List(ctx, deployments, client.InNamespace(dk.Namespace), client.MatchingLabels(deploymentLabels)); err != nil {
		return fmt.Errorf("list deployments: %w", err)
	}

	for _, deploy := range deployments.Items {
		if slices.Contains(keep, deploy.Name) {
			continue
		}

		if err := client.IgnoreNotFound(clt.Delete(ctx, &deploy)); err != nil {
			return fmt.Errorf("delete deployment %s: %w", deploy.Name, err)
		}
	}

	return nil
}
