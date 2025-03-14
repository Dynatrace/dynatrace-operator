package v2

import (
	"fmt"
	"strings"

	containerattr "github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/attributes/container"
	podattr "github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/attributes/pod"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/mounts"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/v2/common/volumes"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/v2/metadata"
	corev1 "k8s.io/api/core/v1"
)

func (wh *Injector) addPodAttributes(request *dtwebhook.MutationRequest) error {
	attrs := podattr.Attributes{
		PodInfo: podattr.PodInfo{
			PodName:       createEnvVarRef(consts.K8sPodNameEnv),
			PodUid:        createEnvVarRef(consts.K8sPodUIDEnv),
			NamespaceName: request.Pod.Namespace,
		},
		ClusterInfo: podattr.ClusterInfo{
			ClusterUId:      request.DynaKube.Status.KubeSystemUUID,
			DTClusterEntity: request.DynaKube.Status.KubernetesClusterMEID,
		},
		UserDefined: map[string]string{
			"k8s.cluster.name": request.DynaKube.Status.KubernetesClusterName, // TODO: make it part of podattr.Attributes
			"k8s.node.name":    createEnvVarRef(consts.K8sNodeNameEnv),        // TODO: make it part of podattr.Attributes
		},
	}

	envs := []corev1.EnvVar{
		{Name: consts.K8sPodNameEnv, ValueFrom: env.NewEnvVarSourceForField("metadata.name")},
		{Name: consts.K8sPodUIDEnv, ValueFrom: env.NewEnvVarSourceForField("metadata.uid")},
		{Name: consts.K8sNodeNameEnv, ValueFrom: env.NewEnvVarSourceForField("spec.nodeName")},
	}

	request.InstallContainer.Env = append(request.InstallContainer.Env, envs...)

	err := metadata.Mutate(wh.metaClient, request, &attrs)
	if err != nil {
		return err
	}

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

func addContainerAttributes(request *dtwebhook.MutationRequest) error {
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
			return err
		}

		request.InstallContainer.Args = append(request.InstallContainer.Args, args...)
	}

	return nil
}

func isInjected(container corev1.Container) bool {
	return mounts.IsIn(container.VolumeMounts, volumes.ConfigVolumeName)
}

func createImageInfo(imageURI string) containerattr.ImageInfo { // TODO: move to bootstrapper repo
	// can't use the name.ParseReference() as that will fill in some defaults if certain things are defined, but we want to preserve the original string value, without any modification. Tried it with a regexp, was worse.
	imageInfo := containerattr.ImageInfo{}

	repoPart := ""

	registrySplit := strings.SplitN(imageURI, "/", 2)
	if len(registrySplit) == 1 {
		repoPart = registrySplit[0]
	} else if len(registrySplit) == 2 {
		imageInfo.Registry = registrySplit[0]
		repoPart = registrySplit[1]
	}

	digestSplit := strings.SplitN(repoPart, "@", 2)
	if len(digestSplit) == 1 {
		repoPart = digestSplit[0]
	} else if len(digestSplit) == 2 {
		imageInfo.ImageDigest = digestSplit[1]
		repoPart = digestSplit[0]
	}

	tagSplit := strings.SplitN(repoPart, ":", 2)
	if len(tagSplit) == 1 {
		imageInfo.Repository = tagSplit[0]
	} else if len(tagSplit) == 2 {
		imageInfo.Tag = tagSplit[1]
		imageInfo.Repository = tagSplit[0]
	}

	return imageInfo
}
