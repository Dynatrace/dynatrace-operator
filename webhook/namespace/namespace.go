package namespace

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/mapper"
	"github.com/Dynatrace/dynatrace-operator/scheme"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func AddNamespaceWebhookToManager(manager ctrl.Manager, ns string) error {
	manager.GetWebhookServer().Register("/label-ns", &webhook.Admission{
		Handler: newNamespaceInjector(ns, manager.GetAPIReader()),
	})
	return nil
}

type namespaceInjector struct {
	logger    logr.Logger
	client    client.Client
	apiReader client.Reader
	namespace string
}

// InjectClient implements the inject.Client interface which allows the manager to inject a kubernetes client into this handler
func (ni *namespaceInjector) InjectClient(clt client.Client) error {
	ni.client = clt
	return nil
}

func (ni *namespaceInjector) Handle(ctx context.Context, request admission.Request) admission.Response {
	if ni.namespace == request.Namespace {
		return admission.Patched("")
	}
	ni.logger.Info("namespace request", "name", request.Name, "namespace", request.Namespace, "operation", request.Operation)
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: request.Namespace}}
	nsMapper := mapper.NewNamespaceMapper(ctx, ni.client, ni.apiReader, ni.namespace, &ns, ni.logger)
	if err := decodeRequestToNamespace(request, &ns); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	if err := nsMapper.MapFromNamespace(); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	ni.logger.Info("Namespace", "labels", ns.Labels)
	return getResponse(&ns, &request)
}

func decodeRequestToNamespace(request admission.Request, namespace *corev1.Namespace) error {
	decoder, err := admission.NewDecoder(scheme.Scheme)
	if err != nil {
		return errors.WithStack(err)
	}

	err = decoder.Decode(request, namespace)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func newNamespaceInjector(ns string, apiReader client.Reader) admission.Handler {
	return &namespaceInjector{
		apiReader: apiReader,
		logger:    logger.NewDTLogger(),
		namespace: ns,
	}
}

func getResponse(ns *corev1.Namespace, req *admission.Request) admission.Response {
	marshaledNamespace, err := json.MarshalIndent(ns, "", "  ")
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledNamespace)
}
