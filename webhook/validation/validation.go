package validation

import (
	"context"
	"fmt"
	"net/http"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/scheme"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	// "k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	exampleApiUrl = "https://ENVIRONMENTID.live.dynatrace.com/api"
)

const (
	errorConflictingOneagentMode = `
The DynaKube's specification tries to use multiple oneagent modes at the same time, which is not supported.
`
	errorConflictingActiveGateSections = `
The DynaKube's specification tries to use the deprecated ActiveGate section(s) alongside the new ActiveGate section, which is not supported.
`

	errorInvalidActiveGateCapability = `
The DynaKube's specification tries to use an invalid capability in ActiveGate section, invalid capability=%s.
Make sure you correctly specify the ActiveGate capabilities in your custom resource.
`

	errorDuplicateActiveGateCapability = `
The DynaKube's specification tries to specify duplicate capabilities in the ActiveGate section, duplicate capability=%s.
Make sure you don't duplicate an Activegate capability in your custom resource.
`

	errorNoApiUrl = `
The DynaKube's specification is missing the API URL or still has the example value set.
Make sure you correctly specify the URL in your custom resource.
`
)

func AddDynakubeValidationWebhookToManager(manager ctrl.Manager) error {
	manager.GetWebhookServer().Register("/validate", &webhook.Admission{
		Handler: newDynakubeValidator(),
	})
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

	dynakube := &dynatracev1beta1.DynaKube{}
	err := decodeRequestToDynakube(request, dynakube)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, errors.WithStack(err))
	}

	if !hasApiUrl(dynakube) {
		validator.logger.Info("requested dynakube has no api url", "name", request.Name, "namespace", request.Namespace)
		return admission.Denied(errorNoApiUrl)
	}

	if hasConflictingOneAgentConfiguration(dynakube) {
		validator.logger.Info("requested dynakube has conflicting one agent configuration", "name", request.Name, "namespace", request.Namespace)
		return admission.Denied(errorConflictingOneagentMode)
	}

	if hasConflictingActiveGateConfiguration(dynakube) {
		validator.logger.Info("requested dynakube has conflicting active gate configuration", "name", request.Name, "namespace", request.Namespace)
		return admission.Denied(errorConflictingActiveGateSections)
	}

	if errMsg := hasInvalidActiveGateCapabilities(dynakube); errMsg != "" {
		validator.logger.Info("requested dynakube has invalid active gate capability", "name", request.Name, "namespace", request.Namespace)
		return admission.Denied(errMsg)
	}

	validator.logger.Info("requested dynakube is valid", "name", request.Name, "namespace", request.Namespace)
	return admission.Allowed("")
}

func hasApiUrl(dynakube *dynatracev1beta1.DynaKube) bool {
	return dynakube.Spec.APIURL != "" && dynakube.Spec.APIURL != exampleApiUrl
}

func hasConflictingOneAgentConfiguration(dynakube *dynatracev1beta1.DynaKube) bool {
	counter := 0
	if dynakube.ApplicationMonitoringMode() {
		counter += 1
	}
	if dynakube.CloudNativeFullstackMode() {
		counter += 1
	}
	if dynakube.ClassicFullStackMode() {
		counter += 1
	}
	if dynakube.HostMonitoringMode() {
		counter += 1
	}
	return counter > 1
}

func hasConflictingActiveGateConfiguration(dynakube *dynatracev1beta1.DynaKube) bool {
	return dynakube.DeprecatedActiveGateMode() && dynakube.ActiveGateMode()
}

func hasInvalidActiveGateCapabilities(dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.ActiveGateMode() {
		capabilities := dynakube.Spec.ActiveGate.Capabilities
		duplicateChecker := map[dynatracev1beta1.CapabilityDisplayName]bool{}
		for _, capability := range capabilities {
			if _, ok := dynatracev1beta1.ActiveGateDisplayNames[capability]; !ok {
				return fmt.Sprintf(errorInvalidActiveGateCapability, capability)
			} else if duplicateChecker[capability] {
				return fmt.Sprintf(errorDuplicateActiveGateCapability, capability)
			}
			duplicateChecker[capability] = true
		}
	}
	return ""
}

// TODO: Implement it to check other dynakubes for conflicting nodeSelectors
//func hasConflictingNodeSelectors(dynakube dynatracev1beta1.DynaKube) bool {
//	infraNodeSelectorMap := dynakube.Spec.InfraMonitoring.NodeSelector
//	classicNodeSelectorMap := dynakube.Spec.ClassicFullStack.NodeSelector
//
//	infraNodeSelector := labels.SelectorFromSet(infraNodeSelectorMap)
//	classicNodeSelector := labels.SelectorFromSet(classicNodeSelectorMap)
//
//	infraNodeSelectorLabels := labels.Set(infraNodeSelectorMap)
//	classicNodeSelectorLabels := labels.Set(classicNodeSelectorMap)
//
//	return infraNodeSelector.Matches(classicNodeSelectorLabels) || classicNodeSelector.Matches(infraNodeSelectorLabels)
//}

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

func newDynakubeValidator() admission.Handler {
	return &dynakubeValidator{
		logger: logger.NewDTLogger(),
	}
}
