package injection

import (
	"fmt"
	"strings"

	containerattr "github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure/attributes/container"
	podattr "github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure/attributes/pod"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/volumes"
	corev1 "k8s.io/api/core/v1"
)

func addPodAttributes(request *dtwebhook.MutationRequest) error {
	attrs := podattr.Attributes{
		PodInfo: podattr.PodInfo{
			PodName:       createEnvVarRef(K8sPodNameEnv),
			PodUID:        createEnvVarRef(K8sPodUIDEnv),
			NodeName:      createEnvVarRef(K8sNodeNameEnv),
			NamespaceName: request.Pod.Namespace,
		},
		ClusterInfo: podattr.ClusterInfo{
			ClusterUID:      request.DynaKube.Status.KubeSystemUUID,
			DTClusterEntity: request.DynaKube.Status.KubernetesClusterMEID,
			ClusterName:     request.DynaKube.Status.KubernetesClusterName,
		},
		UserDefined: map[string]string{},
	}

	envs := []corev1.EnvVar{
		{Name: K8sPodNameEnv, ValueFrom: env.NewEnvVarSourceForField("metadata.name")},
		{Name: K8sPodUIDEnv, ValueFrom: env.NewEnvVarSourceForField("metadata.uid")},
		{Name: K8sNodeNameEnv, ValueFrom: env.NewEnvVarSourceForField("spec.nodeName")},
	}

	request.InstallContainer.Env = append(request.InstallContainer.Env, envs...)

	args, err := podattr.ToArgs(attrs)
	if err != nil {
		return err
	}

	request.InstallContainer.Args = append(request.InstallContainer.Args, args...)

	return nil
}

func createEnvVarRef(envName string) string {
	return fmt.Sprintf("$(%s)", envName)
}

func addContainerAttributes(request *dtwebhook.MutationRequest) (bool, error) {
	attributes := []containerattr.Attributes{}
	for _, c := range request.NewContainers(isInjected) {
		attributes = append(attributes, containerattr.Attributes{
			ImageInfo:     createImageInfo(c.Image),
			ContainerName: c.Name,
		})

		if volumes.IsConfigVolumeMountNeeded(c, request.IsSplitMountsFFEnabled()) {
			volumes.AddConfigVolumeMount(c)
		}

		if volumes.IsOneAgentConfigVolumeMountNeeded(c, request.IsSplitMountsFFEnabled(), request.DynaKube.OneAgent().IsAppInjectionNeeded()) {
			volumes.AddOneAgentConfigVolumeMount(c)
		}

		if volumes.IsEnrichmentConfigVolumeMountNeeded(c, request.IsSplitMountsFFEnabled(), request.DynaKube.MetadataEnrichment().IsEnabled()) {
			volumes.AddEnrichmentConfigVolumeMount(c)
		}
	}

	if len(attributes) > 0 {
		args, err := containerattr.ToArgs(attributes)
		if err != nil {
			return false, err
		}

		request.InstallContainer.Args = append(request.InstallContainer.Args, args...)

		return true, nil
	}

	return false, nil
}

func isInjected(container corev1.Container, splitMountsEnabled bool, isAppInjectionEnabled bool, isMetadataEnrichmentEnabled bool) bool {
	if !volumes.IsConfigVolumeMountNeeded(&container, splitMountsEnabled) &&
		!volumes.IsOneAgentConfigVolumeMountNeeded(&container, splitMountsEnabled, isAppInjectionEnabled) &&
		!volumes.IsEnrichmentConfigVolumeMountNeeded(&container, splitMountsEnabled, isMetadataEnrichmentEnabled) {
		return true
	}

	return false
}

func createImageInfo(imageURI string) containerattr.ImageInfo { // TODO: move to bootstrapper repo
	// can't use the name.ParseReference() as that will fill in some defaults if certain things are defined, but we want to preserve the original string value, without any modification. Tried it with a regexp, was worse.
	imageInfo := containerattr.ImageInfo{}

	registry, repo, found := strings.Cut(imageURI, "/")
	if found {
		imageInfo.Registry = registry
	} else {
		repo = registry
	}

	repo, digest, found := strings.Cut(repo, "@")
	if found {
		imageInfo.ImageDigest = digest
	}

	var tag string

	imageInfo.Repository, tag, found = strings.Cut(repo, ":")
	if found {
		imageInfo.Tag = tag
	}

	return imageInfo
}
