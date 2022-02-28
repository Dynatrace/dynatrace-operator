package validation

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
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

func newDynakubeValidator(apiReader client.Reader, cfg *rest.Config) admission.Handler {
	return &dynakubeValidator{
		apiReader: apiReader,
		cfg:       cfg,
	}
}

func AddDynakubeValidationWebhookToManager(manager ctrl.Manager) error {
	manager.GetWebhookServer().Register("/validate", &webhook.Admission{
		Handler: newDynakubeValidator(manager.GetAPIReader(), manager.GetConfig()),
	})
	return nil
}

// InjectClient implements the inject.Client interface which allows the manager to inject a kubernetes client into this handler
func (validator *dynakubeValidator) InjectClient(clt client.Client) error {
	validator.clt = clt
	return nil
}

func (validator *dynakubeValidator) Handle(_ context.Context, request admission.Request) admission.Response {
	log.Info("validating request", "name", request.Name, "namespace", request.Namespace)

	dynakube := &dynatracev1beta1.DynaKube{}
	err := decodeRequestToDynakube(request, dynakube)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, errors.WithStack(err))
	}
	validationErrors := validator.runValidators(validators, dynakube)
	response := admission.Allowed("")
	if len(validationErrors) > 0 {
		response = admission.Denied(sumErrors(validationErrors))
	}
	warningMessages := validator.runValidators(warnings, dynakube)
	if len(warningMessages) > 0 {
		if hasPreviewWarning(warningMessages) {
			warningMessages = append(warningMessages, basePreviewWarning)
		}
		response = response.WithWarnings(warningMessages...)
	}
	return response
}

func (validator *dynakubeValidator) runValidators(validators []validator, dynakube *dynatracev1beta1.DynaKube) []string {
	results := []string{}
	for _, validate := range validators {
		if errMsg := validate(validator, dynakube); errMsg != "" {
			results = append(results, errMsg)
		}
	}
	return results
}

func sumErrors(validationErrors []string) string {
	summedErrors := fmt.Sprintf("\n%d error(s) found in the Dynakube", len(validationErrors))
	for i, errMsg := range validationErrors {
		summedErrors += fmt.Sprintf("\n %d. %s", i+1, errMsg)
	}
	return summedErrors
}

func decodeRequestToDynakube(request admission.Request, dynakube *dynatracev1beta1.DynaKube) error {
	decoder, err := admission.NewDecoder(scheme.Scheme)
	if err != nil {
		return errors.WithStack(err)
	}

	err = decoder.Decode(request, dynakube)
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
