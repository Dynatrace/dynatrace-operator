package daemonset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/address"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/resources"
	corev1 "k8s.io/api/core/v1"
)

const (
	defaultImageRepo = "public.ecr.aws/dynatrace/dynatrace-k8s-node-config-collector"
	defaultImageTag  = "latest"

	containerName       = "node-config-collector"
	runAs         int64 = 0
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
		Privileged:               address.Of(false),
		AllowPrivilegeEscalation: address.Of(false),
		RunAsUser:                address.Of(runAs),
		RunAsGroup:               address.Of(runAs),
		RunAsNonRoot:             address.Of(false),
		ReadOnlyRootFilesystem:   address.Of(true),
		Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
	}

	return securityContext
}

func getResources(dk dynakube.DynaKube) corev1.ResourceRequirements {
	const (
		defaultCPU = "100m"
		defaultMemory = "128Mi"
	)

	limits := resources.NewResourceList(defaultCPU, defaultMemory)
	requests := resources.NewResourceList(defaultCPU, defaultMemory)

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
