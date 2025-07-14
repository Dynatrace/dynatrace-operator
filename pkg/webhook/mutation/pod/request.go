package pod

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func (wh *webhook) createMutationRequestBase(ctx context.Context, request admission.Request) (*dtwebhook.MutationRequest, error) {
	pod, err := getPodFromRequest(request, wh.decoder)
	if err != nil {
		return nil, err
	}

	namespace, err := getNamespaceFromRequest(ctx, wh.apiReader, request)
	if err != nil {
		return nil, err
	}

	dynakubeName, err := getDynakubeName(*namespace)
	if err != nil && !wh.deployedViaOLM {
		return nil, err
	} else if err != nil {
		// in case of olm deployment, all pods are sent to us
		// but not all of them need to be mutated,
		// therefore their namespace might not have a dynakube assigned
		// in which case we don't need to do anything
		return nil, nil //nolint
	}

	dynakube, err := wh.getDynakube(ctx, dynakubeName)
	if err != nil {
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

func (wh *webhook) getDynakube(ctx context.Context, dynakubeName string) (*dynakube.DynaKube, error) {
	var dk dynakube.DynaKube

	err := wh.apiReader.Get(ctx, client.ObjectKey{Name: dynakubeName, Namespace: wh.webhookNamespace}, &dk)
	if k8serrors.IsNotFound(err) {
		wh.recorder.SendMissingDynaKubeEvent(wh.webhookNamespace, dynakubeName)

		return nil, err
	} else if err != nil {
		return nil, err
	}

	return &dk, nil
}
