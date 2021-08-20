package validation

import (
	"context"
	"net/http"

	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/scheme"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	exampleApiUrl = "https://ENVIRONMENTID.live.dynatrace.com/api"
)

const (
	errorConflictingInfraMonitoringAndClassicNodeSelectors = `
The DynaKubes specifications for infraMonitoring and classicFullStack are conflicting. 
If both are enabled, the nodeSelectors of those specifications must not select the same nodes.
This is due to the infraMonitoring and classicFullStack functionalities being incompatible.
In general, it is advised to use infraMonitoring together with codeModules instead of classicFullStack.
`
	errorNoApiUrl = `
The DynaKube custom resource is missing the API URL or still has the example value set. 
Make sure you correctly specified the URL in your custom resource.
`
)

func AddDynakubeValidationWebhookToManager(manager ctrl.Manager) error {
	manager.GetWebhookServer().Register("/validate", &webhook.Admission{
		Handler: newDynakubeValidator(),
	})
	manager.GetWebhookServer().Register("/healthz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	return nil
}

type dynakubeValidator struct {
	logger logr.Logger
	clt    client.Client
}

// InjectClient implements the inject.Client interface which allows the manager to inject a kubernetes client into this handler
func (validator *dynakubeValidator) InjectClient(clt client.Client) error {
	validator.clt = clt
	return nil
}

func (validator *dynakubeValidator) Handle(_ context.Context, request admission.Request) admission.Response {
	validator.logger.Info("validating request", "name", request.Name, "namespace", request.Namespace)

	var dynakube v1alpha1.DynaKube
	err := decodeRequestToDynakube(request, &dynakube)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, errors.WithStack(err))
	}

	if !hasApiUrl(dynakube) {
		validator.logger.Info("requested dynakube has no api url", "name", request.Name, "namespace", request.Namespace)
		return admission.Denied(errorNoApiUrl)
	}

	if hasConflictingConfiguration(dynakube) {
		validator.logger.Info("requested dynakube has conflicting configuration", "name", request.Name, "namespace", request.Namespace)
		return admission.Denied(errorConflictingInfraMonitoringAndClassicNodeSelectors)
	}

	validator.logger.Info("requested dynakube is valid", "name", request.Name, "namespace", request.Namespace)
	return admission.Allowed("")
}

func hasApiUrl(dynakube v1alpha1.DynaKube) bool {
	return dynakube.Spec.APIURL != "" && dynakube.Spec.APIURL != exampleApiUrl
}

func hasConflictingConfiguration(dynakube v1alpha1.DynaKube) bool {
	return dynakube.Spec.InfraMonitoring.Enabled &&
		dynakube.Spec.ClassicFullStack.Enabled &&
		hasConflictingNodeSelectors(dynakube)
}

func hasConflictingNodeSelectors(dynakube v1alpha1.DynaKube) bool {
	infraNodeSelectorMap := dynakube.Spec.InfraMonitoring.NodeSelector
	classicNodeSelectorMap := dynakube.Spec.ClassicFullStack.NodeSelector

	infraNodeSelector := labels.SelectorFromSet(infraNodeSelectorMap)
	classicNodeSelector := labels.SelectorFromSet(classicNodeSelectorMap)

	infraNodeSelectorLabels := labels.Set(infraNodeSelectorMap)
	classicNodeSelectorLabels := labels.Set(classicNodeSelectorMap)

	return infraNodeSelector.Matches(classicNodeSelectorLabels) || classicNodeSelector.Matches(infraNodeSelectorLabels)
}

func decodeRequestToDynakube(request admission.Request, dynakube *v1alpha1.DynaKube) error {
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

func newDynakubeValidator() admission.Handler {
	return &dynakubeValidator{
		logger: logger.NewDTLogger(),
	}
}
