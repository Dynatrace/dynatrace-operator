package edgeconnect

import (
	"context"
	"net/http"

	"github.com/Dynatrace/dynatrace-operator/src/api/v1alpha1/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/Dynatrace/dynatrace-operator/src/webhook/validation"
	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type edgeconnectValidator struct {
	clt       client.Client
	apiReader client.Reader
	cfg       *rest.Config
}

func newEdgeConnectValidator(apiReader client.Reader, cfg *rest.Config, clt client.Client) admission.Handler {
	return &edgeconnectValidator{
		apiReader: apiReader,
		cfg:       cfg,
		clt:       clt,
	}
}

func AddEdgeConnectValidationWebhookToManager(manager ctrl.Manager) error {
	manager.GetWebhookServer().Register("/validate/edgeconnect", &webhook.Admission{
		Handler: newEdgeConnectValidator(manager.GetAPIReader(), manager.GetConfig(), manager.GetClient()),
	})
	return nil
}

func (validator *edgeconnectValidator) Handle(_ context.Context, request admission.Request) admission.Response {
	log.Info("validating edgeconnect request", "name", request.Name, "namespace", request.Namespace)

	edgeConnect := &edgeconnect.EdgeConnect{}
	err := decodeRequestToEdgeConnect(request, edgeConnect)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, errors.WithStack(err))
	}
	validationErrors := validator.runValidators(validators, edgeConnect)
	response := admission.Allowed("")
	if len(validationErrors) > 0 {
		response = admission.Denied(validation.SumErrors(validationErrors, "EdgeConnect"))
	}
	return response
}

func (validator *edgeconnectValidator) runValidators(validators []validator, edgeConnect *edgeconnect.EdgeConnect) []string {
	results := []string{}
	for _, validate := range validators {
		if errMsg := validate(validator, edgeConnect); errMsg != "" {
			results = append(results, errMsg)
		}
	}
	return results
}

func decodeRequestToEdgeConnect(request admission.Request, edgeConnect *edgeconnect.EdgeConnect) error {
	decoder := admission.NewDecoder(scheme.Scheme)

	err := decoder.Decode(request, edgeConnect)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
