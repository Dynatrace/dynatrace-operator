package v2

import (
	"fmt"

	containerattr "github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/attributes/container"
	podattr "github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/attributes/pod"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/mounts"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/v2/common/volumes"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/v2/metadata"
	"github.com/google/go-containerregistry/pkg/name"
	corev1 "k8s.io/api/core/v1"
)

func (wh *Injector) addPodAttributes(request *dtwebhook.MutationRequest) {
	attr := podattr.Attributes{
		PodInfo: podattr.PodInfo{
			PodName:       createEnvVarRef(consts.K8sPodNameEnv),
			PodUid:        createEnvVarRef(consts.K8sPodUIDEnv),
			NamespaceName: request.Pod.Namespace,
		},
		ClusterInfo: podattr.ClusterInfo{
			ClusterUId:      request.DynaKube.Status.KubeSystemUUID,
			DTClusterEntity: request.DynaKube.Status.KubernetesClusterMEID,
		},
	}

	envs := []corev1.EnvVar{
		{Name: consts.K8sPodNameEnv, ValueFrom: env.NewEnvVarSourceForField("metadata.name")},
		{Name: consts.K8sPodUIDEnv, ValueFrom: env.NewEnvVarSourceForField("metadata.uid")},
	}

	request.InstallContainer.Env = append(request.InstallContainer.Env, envs...)

	metadata.Mutate(wh.metaClient, request, &attr)

	args, err := podattr.ToArgs(attr)
	if err != nil {
		return // TODO
	}

	request.InstallContainer.Args = append(request.InstallContainer.Args, args...)
}

func createEnvVarRef(envName string) string {
	return fmt.Sprintf("$(%s)", envName)
}

func addContainerAttributes(request *dtwebhook.MutationRequest) {
	attributes := []containerattr.Attributes{}
	for _, c := range request.NewContainers(isInjected) {
		attributes = append(attributes, containerattr.Attributes{
			ImageInfo:     createImageInfo(c.Image),
			ContainerName: c.Name,
		})
	}

	if len(attributes) > 0 {
		args, err := containerattr.ToArgs(attributes)
		if err != nil {
			return // TODO fix
		}

		request.InstallContainer.Args = append(request.InstallContainer.Args, args...)
	}
}

func isInjected(container corev1.Container) bool {
	return mounts.IsIn(container.VolumeMounts, volumes.ConfigVolumeName)
}

func createImageInfo(imageURI string) containerattr.ImageInfo { // TODO: move to bootstrapper repo
	ref, _ := name.ParseReference(imageURI)

	tag := ""
	if taggedRef, ok := ref.(name.Tag); ok {
		tag = taggedRef.TagStr()
	}

	digest := ""
	if diggestRef, ok := ref.(name.Digest); ok {
		digest = diggestRef.DigestStr()
	}

	return containerattr.ImageInfo{
		Registry:    ref.Context().RegistryStr(),
		Repository:  ref.Context().RepositoryStr(),
		Tag:         tag,
		ImageDigest: digest,
	}
}
