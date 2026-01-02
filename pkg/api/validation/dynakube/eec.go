package validation

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errorExtensionExecutionControllerImageNotSpecified       = `DynaKube's specification enables extensions, make sure you correctly specify the ExtensionExecutionController image.`
	errorExtensionExecutionControllerInvalidPVCConfiguration = `DynaKube specifies a PVC for the extension controller while ephemeral volume is also enabled. These settings are mutually exclusive, please choose only one.`
	warningConflictingAPIURLForExtensions                    = `You are already using a Dynakube ('%s') that enables extensions. Having multiple Dynakubes with same '.spec.apiUrl' and '.spec.extensions' enabled can have severe side-effects on “sum” and “count” metrics and cause double-billing.`
)

func extensionControllerImage(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if !dk.Extensions().IsAnyEnabled() {
		return ""
	}

	if dk.Spec.Templates.ExtensionExecutionController.ImageRef.Repository == "" || dk.Spec.Templates.ExtensionExecutionController.ImageRef.Tag == "" {
		log.Info("requested dynakube doesn't specify the ExtensionExecutionController image.", "name", dk.Name, "namespace", dk.Namespace)

		return errorExtensionExecutionControllerImageNotSpecified
	}

	return ""
}

func conflictingAPIURLForExtensions(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	if !dk.Extensions().IsAnyEnabled() {
		return ""
	}

	validDynakubes := &dynakube.DynaKubeList{}
	if err := dv.apiReader.List(ctx, validDynakubes, &client.ListOptions{Namespace: dk.Namespace}); err != nil {
		log.Info("error occurred while listing dynakubes", "err", err.Error())

		return ""
	}

	for _, item := range validDynakubes.Items {
		if item.Name == dk.Name {
			continue
		}

		if item.Extensions().IsAnyEnabled() && (dk.APIURL() == item.APIURL()) {
			return fmt.Sprintf(warningConflictingAPIURLForExtensions, item.Name)
		}
	}

	return ""
}

func extensionControllerPVCStorageDevice(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if !dk.Extensions().IsAnyEnabled() {
		return ""
	}

	if extensionControllerMutuallyExclusivePVCSettings(dk) {
		log.Info("requested dynakube specifies mutually exclusive VolumeClaimTemplate settings for ExtensionExecutionController.", "name", dk.Name, "namespace", dk.Namespace)

		return errorExtensionExecutionControllerInvalidPVCConfiguration
	}

	return ""
}

func extensionControllerMutuallyExclusivePVCSettings(dk *dynakube.DynaKube) bool {
	return dk.Spec.Templates.ExtensionExecutionController.UseEphemeralVolume && dk.Spec.Templates.ExtensionExecutionController.PersistentVolumeClaim != nil
}
