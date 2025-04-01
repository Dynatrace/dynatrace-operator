package pod

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/mapper"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
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

	dynakube, err := mapper.GetDynakubeForNamespace(ctx, wh.apiReader, *namespace, wh.webhookNamespace)
	if err != nil {
		var ignored mapper.IgnoredError
		if errors.As(err, &ignored) {
			log.Debug("skipping mutation", "reason", ignored.Error(), "pod", pod.GetGenerateName())
			return nil, nil
		}

		var missing mapper.MissingError
		if errors.As(err, &missing) {
			log.Debug("skipping mutation", "reason", missing.Error(), "pod", pod.GetGenerateName())
			return nil, nil
		}

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
