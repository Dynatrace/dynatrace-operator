package validation

import (
	"context"
	"fmt"
	"net/http"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/mapper"
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
	errorConflictingNamespaceSelector = `
The DynaKube's specification tries to inject into namespaces where another Dynakube already injects into, which is not supported.
Make sure the namespaceSelector doesn't conflict with other Dynakubes namespaceSelector
`

	errorNodeSelectorConflict = `
The DynaKube's specification tries to specify a nodeSelector conflicts with an another Dynakube's nodeSelector, which is not supported.
The conflicting Dynakube: %s
`

	errorNoApiUrl = `
The DynaKube's specification is missing the API URL or still has the example value set.
Make sure you correctly specify the URL in your custom resource.
`
	warningCloudNativeFullStack = `cloudNativeFullStack mode is a BETA feature. Please be aware that it is NOT production ready, and you may run into bugs.`
)

func AddDynakubeValidationWebhookToManager(manager ctrl.Manager) error {
	manager.GetWebhookServer().Register("/validate", &webhook.Admission{
		Handler: newDynakubeValidator(manager.GetAPIReader()),
	})
	return nil
}

type dynakubeValidator struct {
	logger    logr.Logger
	clt       client.Client
	apiReader client.Reader
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

	if validator.hasConflictingNamespaceSelector(dynakube) {
		validator.logger.Info("requested dynakube has conflicting namespaceSelector", "name", request.Name, "namespace", request.Namespace)
		return admission.Denied(errorConflictingNamespaceSelector)
	}

	if errMsg := hasInvalidActiveGateCapabilities(dynakube); errMsg != "" {
		validator.logger.Info("requested dynakube has invalid active gate capability", "name", request.Name, "namespace", request.Namespace)
		return admission.Denied(errMsg)
	}

	if errMsg := hasConflictingNodeSelector(validator.clt, dynakube, validator.logger); errMsg != "" {
		validator.logger.Info("requested dynakube has conflicting nodeSelector", "name", request.Name, "namespace", request.Namespace)
		return admission.Denied(errMsg)
	}

	validator.logger.Info("requested dynakube is valid", "name", request.Name, "namespace", request.Namespace)
	if dynakube.CloudNativeFullstackMode() {
		validator.logger.Info("Dynakube with cloudNativeFullStack was applied, warning was provided.")
		return admission.Allowed("").WithWarnings(warningCloudNativeFullStack)
	}
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

func (validator *dynakubeValidator) hasConflictingNamespaceSelector(dynakube *dynatracev1beta1.DynaKube) bool {
	if !dynakube.NeedAppInjection(){
		return false
	}
	dkMapper := mapper.NewDynakubeMapper(context.TODO(), validator.clt, validator.apiReader, dynakube.Namespace, dynakube, validator.logger)
	_, err := dkMapper.MatchingNamespaces()
	return err != nil
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

func hasConflictingNodeSelector(client client.Client, dynakube *dynatracev1beta1.DynaKube, logger logr.Logger) string {
	if !dynakube.NeedsOneAgent() || dynakube.NodeSelector() == nil {
		return ""
	}
	validDynakubes := &dynatracev1beta1.DynaKubeList{}
	if err := client.List(context.TODO(), validDynakubes); err != nil {
		logger.Info("error occurred while listing dynakubes", "err", err.Error())
		return ""
	}
	for _, item := range validDynakubes.Items {
		nodeSelectorMap := dynakube.NodeSelector()
		validNodeSelectorMap := item.NodeSelector()
		if item.Name != dynakube.Name && hasConflictingMatchLabels(nodeSelectorMap, validNodeSelectorMap) {
			return fmt.Sprintf(errorNodeSelectorConflict, item.Name)
		}
	}
	return ""
}

func hasConflictingMatchLabels(labelMap, otherLabelMap map[string]string) bool {
	if labelMap != nil && otherLabelMap != nil {
		labelSelector := labels.SelectorFromSet(labelMap)
		otherLabelSelector := labels.SelectorFromSet(otherLabelMap)
		labelSelectorLabels := labels.Set(labelMap)
		otherLabelSelectorLabels := labels.Set(otherLabelMap)
		return labelSelector.Matches(otherLabelSelectorLabels) || otherLabelSelector.Matches(labelSelectorLabels)
	}
	return false
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

func newDynakubeValidator(apiReader client.Reader) admission.Handler {
	return &dynakubeValidator{
		logger:    logger.NewDTLogger(),
		apiReader: apiReader,
	}
}
