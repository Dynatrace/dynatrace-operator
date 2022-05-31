package pod_mutator

import (
	"context"
	"fmt"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/src/mapper"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/src/webhook"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func (webhook *podMutatorWebhook) createMutationRequestBase(ctx context.Context, request admission.Request) (*dtwebhook.MutationRequest, error) {
	pod, err := getPodFromRequest(request, webhook.decoder)
	if err != nil {
		return nil, err
	}
	namespace, err := getNamespaceFromRequest(ctx, webhook.apiReader, request)
	if err != nil {
		return nil, err
	}
	dynakubeName, err := getDynakubeName(*namespace)
	if err != nil {
		return nil, err
	}
	dynakube, err := webhook.getDynakube(ctx, dynakubeName)
	if err != nil {
		return nil, err
	}
	mutationRequest := dtwebhook.MutationRequest{
		Context:   ctx,
		Pod:       pod,
		Namespace: namespace,
		DynaKube:  dynakube,
	}
	return &mutationRequest, nil
}

func getPodFromRequest(req admission.Request, decoder admission.Decoder) (*corev1.Pod, error) {
	pod := &corev1.Pod{}
	err := decoder.Decode(req, pod)
	if err != nil {
		log.Error(err, "failed to decode the request for pod injection")
		return nil, err
	}
	return pod, nil
}

func getNamespaceFromRequest(ctx context.Context, apiReader client.Reader, req admission.Request) (*corev1.Namespace, error) {
	var namespace corev1.Namespace

	if err := apiReader.Get(ctx, client.ObjectKey{Name: req.Namespace}, &namespace); err != nil {
		log.Error(err, "failed to query the namespace before pod injection")
		return nil, err
	}
	return &namespace, nil
}

func getDynakubeName(namespace corev1.Namespace) (string, error) {
	dynakubeName, ok := namespace.Labels[mapper.InstanceLabel]
	if !ok {
		var err error
		if !kubesystem.DeployedViaOLM() {
			err = fmt.Errorf("no DynaKube instance set for namespace: %s", namespace.Name)
		}
		return dynakubeName, err
	}
	return dynakubeName, nil
}

func (webhook *podMutatorWebhook) getDynakube(ctx context.Context, dynakubeName string) (*dynatracev1beta1.DynaKube, error) {
	var dk dynatracev1beta1.DynaKube
	err := webhook.apiReader.Get(ctx, client.ObjectKey{Name: dynakubeName, Namespace: webhook.webhookNamespace}, &dk)
	if k8serrors.IsNotFound(err) {
		webhook.recorder.sendMissingDynaKubeEvent(webhook.webhookNamespace, dynakubeName)
		return nil, err
	} else if err != nil {
		return nil, err
	}
	return &dk, nil
}
