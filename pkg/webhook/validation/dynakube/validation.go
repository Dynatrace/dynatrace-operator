package dynakube

import (
	"context"
	"net/http"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/validation"
	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type dynakubeValidator struct {
	clt       client.Client
	apiReader client.Reader
	cfg       *rest.Config
}

var _ admission.Handler = &dynakubeValidator{}

func newDynakubeValidator(clt client.Client, apiReader client.Reader, cfg *rest.Config) admission.Handler {
	return &dynakubeValidator{
		apiReader: apiReader,
		cfg:       cfg,
		clt:       clt,
	}
}

func AddDynakubeValidationWebhookToManager(manager ctrl.Manager) error {
	log.Info("Register Validator to /validate")
	manager.GetWebhookServer().Register("/validate", &webhook.Admission{
		Handler: newDynakubeValidator(manager.GetClient(), manager.GetAPIReader(), manager.GetConfig()),
	})
	return nil
}

func (validator *dynakubeValidator) Handle(ctx context.Context, request admission.Request) admission.Response {
	log.Info("validating request", "name", request.Name, "namespace", request.Namespace)

	dynakube := &dynatracev1beta1.DynaKube{}
	err := decodeRequestToDynakube(request, dynakube)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, errors.WithStack(err))
	}
	validationErrors := validator.runValidators(ctx, validators, dynakube)
	response := admission.Allowed("")
	if len(validationErrors) > 0 {
		response = admission.Denied(validation.SumErrors(validationErrors, "Dynakube"))
	}
	warningMessages := validator.runValidators(ctx, warnings, dynakube)
	if len(warningMessages) > 0 {
		if hasPreviewWarning(warningMessages) {
			warningMessages = append(warningMessages, basePreviewWarning)
		}
		response = response.WithWarnings(warningMessages...)
	}
	return response
}

func (validator *dynakubeValidator) runValidators(ctx context.Context, validators []validator, dynakube *dynatracev1beta1.DynaKube) []string {
	results := []string{}
	for _, validate := range validators {
		if errMsg := validate(ctx, validator, dynakube); errMsg != "" {
			results = append(results, errMsg)
		}
	}
	return results
}

func decodeRequestToDynakube(request admission.Request, dynakube *dynatracev1beta1.DynaKube) error {
	log.Info("decodeRequestToDynakube", "request", request)
	decoder := admission.NewDecoder(scheme.Scheme)

	err := decoder.Decode(request, dynakube)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func hasPreviewWarning(warnings []string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, "PREVIEW") {
			return true
		}
	}
	return false
}
