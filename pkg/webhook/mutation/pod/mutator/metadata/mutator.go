// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package metadata

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/k8sinit/configure/attributes/container"
	"github.com/Dynatrace/dynatrace-operator/cmd/bootstrapper"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sresource"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/arg"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/attributes"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/volumes"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Mutator struct {
	metaClient client.Client
}

func NewMutator(metaClient client.Client) dtwebhook.Mutator {
	return &Mutator{
		metaClient: metaClient,
	}
}

func (mut *Mutator) IsEnabled(_ context.Context, request *dtwebhook.BaseRequest) bool {
	enabledOnPod := maputils.GetFieldBool(request.Pod.Annotations, AnnotationInject,
		request.DynaKube.FF().IsAutomaticInjection())
	enabledOnDynakube := request.DynaKube.MetadataEnrichment().IsEnabled()

	matchesNamespace := true // if no namespace selector is configured, we just pass set this to true

	if request.DynaKube.MetadataEnrichment().GetNamespaceSelector().Size() > 0 {
		selector, _ := metav1.LabelSelectorAsSelector(request.DynaKube.MetadataEnrichment().GetNamespaceSelector())

		matchesNamespace = selector.Matches(labels.Set(request.Namespace.Labels))
	}

	return matchesNamespace && enabledOnPod && enabledOnDynakube
}

func (mut *Mutator) IsInjected(_ context.Context, request *dtwebhook.BaseRequest) bool {
	return maputils.GetFieldBool(request.Pod.Annotations, AnnotationInjected, false)
}

func initResources(dk dynakube.DynaKube) corev1.ResourceRequirements {
	if custom := dk.MetadataEnrichment().GetInitResources(); custom != nil {
		return *custom
	}

	return corev1.ResourceRequirements{
		Requests: k8sresource.NewResourceList("30m", "30Mi"),
		Limits:   k8sresource.NewResourceList("100m", "60Mi"),
	}
}

func (mut *Mutator) Mutate(request *dtwebhook.MutationRequest) error {
	_, log := logd.NewFromContext(request.Context, "metadata-enrichment")
	log.Info("adding metadata-enrichment to pod", "name", request.PodName())

	request.InstallContainer.Resources = initResources(request.DynaKube)

	attrs, err := attributes.NewPodAttributes(request.Context, *request.BaseRequest, mut.metaClient)
	if err != nil {
		return dtwebhook.MutatorError{
			Err:      errors.WithStack(err),
			Annotate: setNotInjectedAnnotationFunc(OwnerLookupFailedReason),
		}
	}

	if oneagent.IsEnabled(request.BaseRequest) {
		attrs.SetDynakubeAttributes(request.Context, request.DynaKube.OneAgent().GetResourceAttributes())
	} else {
		attrs.SetDynakubeAttributes(request.Context, request.DynaKube.GetResourceAttributes())
	}

	withDeprecatedAttributesArg := arg.Arg{
		Name:  bootstrapper.EnableAttributesDTKubernetesFlag,
		Value: strconv.FormatBool(request.DynaKube.FF().EnableAttributesDTKubernetes()),
	}

	args := attrs.Convert(func(key, value string) string {
		if key == "" || value == "" {
			return ""
		}

		return attributes.ToArg(key, value)
	})

	request.InstallContainer.Args = append(request.InstallContainer.Args, arg.ConvertArgsToStrings([]arg.Arg{withDeprecatedAttributesArg})...)
	request.InstallContainer.Args = append(request.InstallContainer.Args, args...)

	request.InstallContainer.Env = append(request.InstallContainer.Env, attrs.GetPodEnvVars()...)

	turnOnMetadataEnrichment(request)

	setInjectedAnnotation(request.Pod)

	request.AnnotationWriter = attrs

	_, err = AddContainerAttributes(request.BaseRequest, request.InstallContainer)
	if err != nil {
		return err
	}

	return nil
}

func turnOnMetadataEnrichment(request *dtwebhook.MutationRequest) {
	request.InstallContainer.Args = append(request.InstallContainer.Args, arg.ConvertArgsToStrings([]arg.Arg{{Name: bootstrapper.MetadataEnrichmentFlag}})...)
}

func (mut *Mutator) Reinvoke(_ context.Context, _ *dtwebhook.ReinvocationRequest) bool {
	return false
}

func setInjectedAnnotation(pod *corev1.Pod) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	pod.Annotations[AnnotationInjected] = "true"
	delete(pod.Annotations, AnnotationReason)
}

func setNotInjectedAnnotationFunc(reason string) func(*corev1.Pod) {
	return func(pod *corev1.Pod) {
		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string)
		}

		pod.Annotations[AnnotationInjected] = "false"
		pod.Annotations[AnnotationReason] = reason
	}
}

func AddContainerAttributes(request *dtwebhook.BaseRequest, installContainer *corev1.Container) (bool, error) {
	containers := request.NewContainers(isInjected)
	if len(containers) > 0 {
		args := make([]string, 0)

		for _, c := range containers {
			contInfos := *attributes.NewContainerInfo(*c)

			json, err := contInfos.ToJSON()
			if err != nil {
				return false, err
			}

			args = append(args, fmt.Sprintf("--%s=%s", container.Flag, json))

			volumes.AddConfigVolumeMount(c, request)
		}

		installContainer.Args = append(installContainer.Args, args...)

		return true, nil
	}

	return false, nil
}

func isInjected(container corev1.Container, request *dtwebhook.BaseRequest) bool {
	if request.IsSplitMountsEnabled() {
		if (request.DynaKube.OneAgent().IsAppInjectionNeeded() && !volumes.HasSplitOneAgentMounts(&container)) ||
			(request.DynaKube.MetadataEnrichment().IsEnabled() && !volumes.HasSplitEnrichmentMounts(&container)) {
			return false
		}

		return true
	} else {
		return volumes.HasCommonConfigVolumeMounts(&container)
	}
}
