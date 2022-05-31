package namespace_mutator

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Dynatrace/dynatrace-operator/src/mapper"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func AddNamespaceMutationWebhookToManager(manager ctrl.Manager, ns string) error {
	manager.GetWebhookServer().Register("/label-ns", &webhook.Admission{
		Handler: newNamespaceMutator(ns, manager.GetAPIReader()),
	})
	return nil
}

// namespaceMutator adds the necessary label to namespaces that match a dynakubes namespace selector
type namespaceMutator struct {
	client    client.Client
	apiReader client.Reader
	namespace string
}

// InjectClient implements the inject.Client interface which allows the manager to inject a kubernetes client into this handler
func (nm *namespaceMutator) InjectClient(clt client.Client) error {
	nm.client = clt
	return nil
}

// Handle does the mapping between the namespace and dynakube from the namespace's side.
// There are 2 special cases:
// 1. ignore the webhook's namespace: this is necessary because if we want to monitor the whole cluster
//    we would tag our own namespace which would cause the podInjector webhook to inject into our pods which can cause issues. (infra-monitoring pod injected into == bad)
// 2. if the namespace was updated by the operator => don't do the mapping: we detect this using an annotation, we do this because the operator also does the mapping
//    but from the dynakube's side (during dynakube reconcile) and we don't want to repeat ourselves. So we just remove the annotation.
func (nm *namespaceMutator) Handle(ctx context.Context, request admission.Request) admission.Response {
	if nm.namespace == request.Namespace {
		return admission.Patched("")
	}

	log.Info("namespace request", "namespace", request.Name, "operation", request.Operation)
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: request.Namespace}}
	nsMapper := mapper.NewNamespaceMapper(ctx, nm.client, nm.apiReader, nm.namespace, &ns)
	if err := decodeRequestToNamespace(request, &ns); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if _, ok := ns.Annotations[mapper.UpdatedViaDynakubeAnnotation]; ok {
		log.Info("checking namespace labels not necessary", "namespace", request.Name)
		delete(ns.Annotations, mapper.UpdatedViaDynakubeAnnotation)
		return getResponseForNamespace(&ns, &request)
	}

	log.Info("checking namespace labels", "namespace", request.Name)
	updatedNamespace, err := nsMapper.MapFromNamespace()
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	if !updatedNamespace {
		return admission.Patched("")
	}
	log.Info("namespace", "labels", ns.Labels)
	return getResponseForNamespace(&ns, &request)
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

func newNamespaceMutator(ns string, apiReader client.Reader) admission.Handler {
	return &namespaceMutator{
		apiReader: apiReader,
		namespace: ns,
	}
}

func getResponseForNamespace(ns *corev1.Namespace, req *admission.Request) admission.Response {
	marshaledNamespace, err := json.MarshalIndent(ns, "", "  ")
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledNamespace)
}
