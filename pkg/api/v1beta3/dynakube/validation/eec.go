package validation

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"golang.org/x/net/context"
)

const (
	errorExtensionExecutionControllerImageNotSpecified = `DynaKube's specification enables the Prometheus feature, make sure you correctly specify the ExtensionExecutionController image.`
)

func extensionControllerImage(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if !dk.IsExtensionsEnabled() {
		return ""
	}

	if dk.Spec.Templates.ExtensionExecutionController.ImageRef.Repository == "" || dk.Spec.Templates.ExtensionExecutionController.ImageRef.Tag == "" {
		log.Info("requested dynakube doesn't specify the ExtensionExecutionController image.", "name", dk.Name, "namespace", dk.Namespace)

		return errorExtensionExecutionControllerImageNotSpecified
	}

	return ""
}
