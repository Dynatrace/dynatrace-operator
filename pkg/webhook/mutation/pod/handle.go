package pod

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/bootstrapperconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/container"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (wh *webhook) handle(mutationRequest *dtwebhook.MutationRequest) error {
	wh.recorder.Setup(mutationRequest)

	if !wh.isInputSecretPresent(mutationRequest, bootstrapperconfig.GetSourceConfigSecretName(mutationRequest.DynaKube.Name), consts.BootstrapperInitSecretName) {
		return nil
	}

	if mutationRequest.DynaKube.IsAGCertificateNeeded() || mutationRequest.DynaKube.Spec.TrustedCAs != "" {
		if !wh.isInputSecretPresent(mutationRequest, bootstrapperconfig.GetSourceCertsSecretName(mutationRequest.DynaKube.Name), consts.BootstrapperInitCertsSecretName) {
			return nil
		}
	}

	if wh.isInjected(mutationRequest) {
		if wh.handlePodReinvocation(mutationRequest) {
			log.Info("reinvocation policy applied", "podName", mutationRequest.PodName())
			wh.recorder.SendPodUpdateEvent()

			return nil
		}

		log.Info("no change, all containers already injected", "podName", mutationRequest.PodName())

		return nil
	} else {
		mutated, err := wh.handlePodMutation(mutationRequest)
		if err != nil {
			return err
		}

		if !mutated {
			setNotInjectedAnnotations(mutationRequest, NoMutationNeededReason)

			return nil
		}
	}

	setDynatraceInjectedAnnotation(mutationRequest)

	log.Info("injection finished for pod", "podName", mutationRequest.PodName(), "namespace", mutationRequest.Namespace.Name)

	return nil
}

func (wh *webhook) isInjected(mutationRequest *dtwebhook.MutationRequest) bool {
	installContainer := container.FindInitContainerInPodSpec(&mutationRequest.Pod.Spec, dtwebhook.InstallContainerName)
	if installContainer != nil {
		log.Info("Dynatrace init-container already present, skipping mutation, doing reinvocation", "containerName", dtwebhook.InstallContainerName)

		return true
	}

	return false
}

func (wh *webhook) handlePodMutation(mutationRequest *dtwebhook.MutationRequest) (bool, error) {
	mutationRequest.InstallContainer = wh.createInitContainerBase(mutationRequest.Pod, mutationRequest.DynaKube)

	var mutated bool

	if wh.oaMutator.IsEnabled(mutationRequest.BaseRequest) {
		err := wh.oaMutator.Mutate(mutationRequest)
		if err != nil {
			return false, err
		}

		mutated = true
	}

	if wh.metaMutator.IsEnabled(mutationRequest.BaseRequest) {
		err := wh.metaMutator.Mutate(mutationRequest)
		if err != nil {
			return false, err
		}

		mutated = true
	}

	if mutated {
		_, err := addContainerAttributes(mutationRequest)
		if err != nil {
			return false, err
		}

		addPodAttributes(mutationRequest)

		addInitContainerToPod(mutationRequest.Pod, mutationRequest.InstallContainer)
		wh.recorder.SendPodInjectEvent()
	}

	return mutated, nil
}

func (wh *webhook) handlePodReinvocation(mutationRequest *dtwebhook.MutationRequest) bool {
	mutationRequest.InstallContainer = container.FindInitContainerInPodSpec(&mutationRequest.Pod.Spec, dtwebhook.InstallContainerName)

	// metadata enrichment does not need to be reinvoked, addContainerAttributes() does what is needed
	hasNewContainers, err := addContainerAttributes(mutationRequest)
	if err != nil {
		log.Error(err, "error during reinvocation for updating the init-container, failed to update container-attributes on the init container")

		return false
	}

	var oaUpdated bool
	if wh.oaMutator.IsEnabled(mutationRequest.BaseRequest) {
		oaUpdated = wh.oaMutator.Reinvoke(mutationRequest.ToReinvocationRequest())
	}

	return hasNewContainers || oaUpdated
}

func (wh *webhook) isInputSecretPresent(mutationRequest *dtwebhook.MutationRequest, sourceSecretName, targetSecretName string) bool {
	err := wh.replicateSecret(mutationRequest, sourceSecretName, targetSecretName)
	if k8serrors.IsNotFound(err) {
		log.Info(fmt.Sprintf("unable to copy source of %s as it is not available, injection not possible", sourceSecretName), "pod", mutationRequest.PodName())

		setNotInjectedAnnotations(mutationRequest, NoBootstrapperConfigReason)

		return false
	}

	if err != nil {
		log.Error(err, fmt.Sprintf("unable to verify, if %s is available, injection not possible", sourceSecretName))

		setNotInjectedAnnotations(mutationRequest, NoBootstrapperConfigReason)

		return false
	}

	return true
}

func (wh *webhook) replicateSecret(mutationRequest *dtwebhook.MutationRequest, sourceSecretName, targetSecretName string) error {
	var initSecret corev1.Secret

	secretObjectKey := client.ObjectKey{Name: targetSecretName, Namespace: mutationRequest.Namespace.Name}

	err := wh.apiReader.Get(mutationRequest.Context, secretObjectKey, &initSecret)
	if k8serrors.IsNotFound(err) {
		log.Info(targetSecretName+" is not available, trying to replicate", "pod", mutationRequest.PodName())

		return bootstrapperconfig.Replicate(mutationRequest.Context, mutationRequest.DynaKube, secret.Query(wh.kubeClient, wh.apiReader, log), sourceSecretName, targetSecretName, mutationRequest.Namespace.Name)
	}

	return nil
}

func setDynatraceInjectedAnnotation(mutationRequest *dtwebhook.MutationRequest) {
	if mutationRequest.Pod.Annotations == nil {
		mutationRequest.Pod.Annotations = make(map[string]string)
	}

	mutationRequest.Pod.Annotations[dtwebhook.AnnotationDynatraceInjected] = "true"
	delete(mutationRequest.Pod.Annotations, dtwebhook.AnnotationDynatraceReason)
}

func setNotInjectedAnnotations(mutationRequest *dtwebhook.MutationRequest, reason string) {
	if mutationRequest.Pod.Annotations == nil {
		mutationRequest.Pod.Annotations = make(map[string]string)
	}

	mutationRequest.Pod.Annotations[dtwebhook.AnnotationDynatraceInjected] = "false"
	mutationRequest.Pod.Annotations[dtwebhook.AnnotationDynatraceReason] = reason
}
