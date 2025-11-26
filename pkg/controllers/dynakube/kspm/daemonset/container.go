package daemonset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sresource"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

const (
	defaultImageRepo = "public.ecr.aws/dynatrace/dynatrace-k8s-node-config-collector"
	defaultImageTag  = "latest"

	containerName       = "node-config-collector"
	runAs         int64 = 65532
)

func getContainer(dk dynakube.DynaKube, tenantUUID string) corev1.Container {
	securityContext := getSecurityContext()

	container := corev1.Container{
		Name:            containerName,
		Image:           dk.KSPM().ImageRef.StringWithDefaults(defaultImageRepo, defaultImageTag),
		ImagePullPolicy: corev1.PullAlways,
		VolumeMounts:    getMounts(dk),
		Env:             getEnvs(dk, tenantUUID),
		SecurityContext: &securityContext,
		Resources:       getResources(dk),
	}

	return container
}

func getSecurityContext() corev1.SecurityContext {
	securityContext := corev1.SecurityContext{
		Privileged:               ptr.To(false),
		AllowPrivilegeEscalation: ptr.To(false),
		RunAsUser:                ptr.To(runAs),
		RunAsGroup:               ptr.To(runAs),
		RunAsNonRoot:             ptr.To(true),
		ReadOnlyRootFilesystem:   ptr.To(true),
		Capabilities:             &corev1.Capabilities{Add: []corev1.Capability{"DAC_OVERRIDE"}, Drop: []corev1.Capability{"ALL"}},
	}

	return securityContext
}

func getResources(dk dynakube.DynaKube) corev1.ResourceRequirements {
	const (
		defaultCPU    = "100m"
		defaultMemory = "128Mi"
	)

	limits := k8sresource.NewResourceList(defaultCPU, defaultMemory)
	requests := k8sresource.NewResourceList(defaultCPU, defaultMemory)

	if dk.KSPM().Resources.Limits != nil {
		limits = dk.KSPM().Resources.Limits
	}

	if dk.KSPM().Resources.Requests != nil {
		requests = dk.KSPM().Resources.Requests
	}

	return corev1.ResourceRequirements{
		Requests: requests,
		Limits:   limits,
	}
}
