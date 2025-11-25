package injection

import (
	"fmt"
	"strings"

	containerattr "github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure/attributes/container"
	podattr "github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure/attributes/pod"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8smount"
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

	setDeprecatedAttributes(&attrs)

	envs := []corev1.EnvVar{
		{Name: K8sPodNameEnv, ValueFrom: k8senv.NewSourceForField("metadata.name")},
		{Name: K8sPodUIDEnv, ValueFrom: k8senv.NewSourceForField("metadata.uid")},
		{Name: K8sNodeNameEnv, ValueFrom: k8senv.NewSourceForField("spec.nodeName")},
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

		volumes.AddConfigVolumeMount(c, request.BaseRequest)
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

func isInjected(container corev1.Container, request *dtwebhook.BaseRequest) bool {
	if request.IsSplitMountsEnabled() {
		if request.DynaKube.OneAgent().IsAppInjectionNeeded() && !k8smount.ContainsPath(container.VolumeMounts, volumes.ConfigMountPathOneAgent) ||
			request.DynaKube.MetadataEnrichment().IsEnabled() && !k8smount.ContainsPath(container.VolumeMounts, volumes.ConfigMountPathEnrichment) {
			return false
		}

		return true
	} else {
		return k8smount.ContainsPath(container.VolumeMounts, volumes.ConfigMountPath)
	}
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
