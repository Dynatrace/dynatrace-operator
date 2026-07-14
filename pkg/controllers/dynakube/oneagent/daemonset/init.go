package daemonset

import (
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/sanitize"
	corev1 "k8s.io/api/core/v1"
)

const (
	initContainerName = "dynatrace-operator"
)

func (b *builder) initContainerSpec() corev1.Container {
	return corev1.Container{
		Image:           os.Getenv(k8senv.DTOperatorImageEnvName),
		ImagePullPolicy: b.dk.OneAgent().GetImagePullPolicy(),
		Name:            initContainerName,
		Env:             b.initContainerEnvVars(),
		Args:            b.initContainerArguments(),
		VolumeMounts:    b.initContainerVolumeMounts(),
		SecurityContext: b.initContainerSecurityContext(),
		Resources:       b.hostInjectSpec.OneAgentResources,
	}
}

func (b *builder) initContainerEnvVars() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name: dtNodeName,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		},
	}
}

func (b *builder) initContainerArguments() []string {
	attributes := []string{
		"k8s.cluster.name=" + b.dk.Status.KubernetesClusterName,
		"k8s.cluster.uid=" + b.dk.Status.KubeSystemUUID,
		"k8s.node.name=$(DT_K8S_NODE_NAME)",
	}

	if b.dk.Status.KubernetesClusterMEID != "" {
		attributes = append(attributes, "dt.entity.kubernetes_cluster="+b.dk.Status.KubernetesClusterMEID)
	}

	resourceAttrs := b.dk.OneAgent().GetResourceAttributes()
	for _, k := range slices.Sorted(maps.Keys(resourceAttrs)) {
		attributes = append(attributes, k+"="+resourceAttrs[k])
	}

	initArgs := []string{
		"generate-metadata",
		"--file",
		nodeMetadataFilePath,
		"--attributes",
		strings.Join(attributes, ","),
	}

	return sanitize.CommandLineArgs(initArgs)
}

func (b *builder) initContainerVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      nodeMetadataVolumeName,
			MountPath: nodeMetadataFolderPath,
			ReadOnly:  false,
		},
	}
}

func (b *builder) initContainerSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		Privileged:               new(false),
		AllowPrivilegeEscalation: new(false),
		RunAsNonRoot:             new(true),
		RunAsUser:                new(userGroupID),
		RunAsGroup:               new(userGroupID),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				"ALL",
			},
		},
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
		ReadOnlyRootFilesystem: new(true),
	}
}
