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
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	probePort                = int32(8080)
	livenessProbePath        = "/health/live"
	readinessProbePath       = "/health/ready"
	userGroupID        int64 = 1000
	// Keep in sync with helm chart
	defaultServiceAccount = "dynatrace-database-extensions-executor"
	// Must contain the ID specified in the DynaKube CR.
	executorIDLabelKey = "extensions.dynatrace.com/executor.id"

	tmpVolumeName         = "tmp-data"
	tmpMountPath          = "/tmp"
	tokenVolumeName       = "auth-token"
	tokenMountPath        = "/var/run/dynatrace/executor/token"
	certsVolumeName       = "https-certs"
	certsMountPath        = "/var/ssl-certs/dynatrace"
	customCertsVolumeName = "custom-certs"
	customCertsMountPath  = "/var/ssl-certs/user"
	customCertsFileName   = "custom.crt"
)

// ListDeployments returns a list of database datasource deployments that are managed by the DynaKube.
func ListDeployments(ctx context.Context, clt client.Reader, dk *dynakube.DynaKube) ([]appsv1.Deployment, error) {
	deployments := &appsv1.DeploymentList{}

	// We have to build labels from an empty DB spec to allow cleaning up orphans (without having to delete the DynaKube).
	deploymentLabels, _, _ := buildAllLabels(dk, extensions.DatabaseSpec{})

	if err := clt.List(ctx, deployments, client.InNamespace(dk.Namespace), sanitizedListLabels(deploymentLabels)); err != nil {
		return nil, fmt.Errorf("list deployments: %w", err)
	}

	return deployments.Items, nil
}

// Returns labels for deployment, deployment selector and deployment pod template in that order.
// Do NOT modify maps produced by this function.
func buildAllLabels(dk *dynakube.DynaKube, dbSpec extensions.DatabaseSpec) (map[string]string, map[string]string, map[string]string) {
	appLabels := labels.NewAppLabels(labels.DatabaseDatasourceLabel, dk.Name, labels.DatabaseDatasourceLabel, dk.Spec.Templates.DatabaseExecutor.ImageRef.Tag)

	deploymentLabels := appLabels.BuildLabels()
	matchLabels := appLabels.BuildMatchLabels()
	podLabels := deploymentLabels
	podLabels[executorIDLabelKey] = dbSpec.ID
	podLabels[consts.DatasourceLabelKey] = consts.DatabaseDatasourceLabelValue

	if dbSpec.Labels != nil {
		// Always merge into user-provided labels to ensure they don't overwrite our own.
		temp := maps.Clone(dbSpec.Labels)
		maps.Copy(temp, podLabels)
		podLabels = temp
	}

	// Reuse pod labels for deployment
	return deploymentLabels, matchLabels, podLabels
}

func buildServiceAccountName(dbSpec extensions.DatabaseSpec) string {
	if dbSpec.ServiceAccountName != "" {
		return dbSpec.ServiceAccountName
	}

	return defaultServiceAccount
}

func buildContainer(dk *dynakube.DynaKube, dbSpec extensions.DatabaseSpec) corev1.Container {
	pullPolicy := corev1.PullIfNotPresent
	if dk.Spec.Templates.DatabaseExecutor.ImageRef.Tag == "latest" {
		// For initial testing latest image is used, so let runtime pull updates if they're available.
		// Maybe move this into the imageRef, e.g. imageRef.PullPolicy()
		pullPolicy = corev1.PullAlways
	}

	container := corev1.Container{
		Name:            "database-datasource",
		Image:           dk.Spec.Templates.DatabaseExecutor.ImageRef.String(),
		ImagePullPolicy: pullPolicy,
		Args:            buildContainerArgs(dk),
		Env:             buildContainerEnvs(),
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: livenessProbePath,
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
					Path: readinessProbePath,
					Port: intstr.IntOrString{IntVal: probePort},
				},
			},
			InitialDelaySeconds: 10,
			PeriodSeconds:       5,
			TimeoutSeconds:      2,
			FailureThreshold:    3,
			SuccessThreshold:    1,
		},
		Resources:       buildContainerResources(dbSpec.Resources),
		SecurityContext: buildContainerSecurityContext(),
		VolumeMounts:    buildVolumeMounts(dk, dbSpec),
	}

	return container
}

func buildContainerArgs(dk *dynakube.DynaKube) []string {
	return []string{
		"--podid=$(POD_UID)",
		fmt.Sprintf("--url=https://%s:%d", dk.Extensions().GetServiceNameFQDN(), consts.ExtensionsDatasourceTargetPort),
		"--idtoken=" + tokenMountPath + "/" + tokenVolumeName,
	}
}

func buildContainerEnvs() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name: "POD_UID",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.uid",
				},
			},
		},
	}
}

func buildVolumeMounts(dk *dynakube.DynaKube, dbSpec extensions.DatabaseSpec) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      tmpVolumeName,
			MountPath: tmpMountPath,
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

	if dk.Spec.TrustedCAs != "" {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      customCertsVolumeName,
			MountPath: customCertsMountPath,
			ReadOnly:  true,
		})
	}

	return append(volumeMounts, dbSpec.VolumeMounts...)
}

func buildVolumes(dk *dynakube.DynaKube, dbSpec extensions.DatabaseSpec) []corev1.Volume {
	volumes := []corev1.Volume{
		{
			Name: tmpVolumeName,
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
					Items: []corev1.KeyToPath{
						{
							Key:  consts.TLSCrtDataName,
							Path: consts.TLSCrtDataName,
						},
					},
				},
			},
		},
	}

	if dk.Spec.TrustedCAs != "" {
		volumes = append(volumes, corev1.Volume{
			Name: customCertsVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: dk.Spec.TrustedCAs,
					},
					Items: []corev1.KeyToPath{
						{
							Key:  dynakube.TrustedCAKey,
							Path: customCertsFileName,
						},
					},
				},
			},
		})
	}

	return append(volumes, dbSpec.Volumes...)
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
		FSGroup: ptr.To(userGroupID),
	}
}

func buildContainerSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		Privileged:               ptr.To(false),
		AllowPrivilegeEscalation: ptr.To(false),
		ReadOnlyRootFilesystem:   ptr.To(true),
		RunAsNonRoot:             ptr.To(true),
		RunAsGroup:               ptr.To(userGroupID),
		RunAsUser:                ptr.To(userGroupID),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				"ALL",
			},
		},
	}
}

func deleteDeployments(ctx context.Context, clt client.Client, dk *dynakube.DynaKube, keep []string) error {
	deployments, err := ListDeployments(ctx, clt, dk)
	if err != nil {
		return err
	}

	for _, deploy := range deployments {
		if slices.Contains(keep, deploy.Name) {
			continue
		}

		if err := clt.Delete(ctx, &deploy); err != nil {
			if !k8serrors.IsNotFound(err) {
				return fmt.Errorf("delete deployment %s: %w", deploy.Name, err)
			}

			return nil
		}

		log.Info("deleted deployment", "name", deploy.Name)
	}

	return nil
}

func sanitizedListLabels(deploymentLabels map[string]string) client.MatchingLabels {
	// Remove instance-specific keys to ensure we get all related deployments.
	delete(deploymentLabels, executorIDLabelKey)
	delete(deploymentLabels, consts.DatasourceLabelKey)
	delete(deploymentLabels, labels.AppVersionLabel)

	return deploymentLabels
}
