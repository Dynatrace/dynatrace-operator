package pod_mutator

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dtotel"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	webhookotel "github.com/Dynatrace/dynatrace-operator/pkg/webhook/otel"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func (webhook *podMutatorWebhook) createMutationRequestBase(ctx context.Context, request admission.Request) (*dtwebhook.MutationRequest, error) {
	ctx, span := dtotel.StartSpan(ctx, webhookotel.Tracer())
	defer span.End()

	pod, err := getPodFromRequest(request, webhook.decoder)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	namespace, err := getNamespaceFromRequest(ctx, webhook.apiReader, request)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	dynakubeName, err := getDynakubeName(*namespace)
	if err != nil && !webhook.deployedViaOLM {
		span.RecordError(err)
		return nil, err
	} else if err != nil {
		// in case of olm deployment, all pods are sent to us
		// but not all of them need to be mutated,
		// therefore their namespace might not have a dynakube assigned
		// in which case we don't need to do anything
		span.RecordError(err)
		return nil, nil // nolint:nilerr
	}
	dynakube, err := webhook.getDynakube(ctx, dynakubeName)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	mutationRequest := dtwebhook.NewMutationRequest(ctx, *namespace, nil, pod, *dynakube)
	return mutationRequest, nil
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
	dynakubeName, ok := namespace.Labels[dtwebhook.InjectionInstanceLabel]
	if !ok {
		return "", errors.Errorf("no DynaKube instance set for namespace: %s", namespace.Name)
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
