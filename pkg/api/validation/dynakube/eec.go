// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errorExtensionExecutionControllerImageNotSpecified       = `DynaKube's specification enables extensions, make sure you correctly specify the ExtensionExecutionController image.`
	errorExtensionExecutionControllerInvalidPVCConfiguration = `DynaKube specifies a PVC for the extension controller while ephemeral volume is also enabled. These settings are mutually exclusive, please choose only one.`
	warningConflictingAPIURLForExtensions                    = `You are already using a Dynakube ('%s') that enables extensions. Having multiple Dynakubes with same '.spec.apiUrl' and '.spec.extensions' enabled can have severe side-effects on “sum” and “count” metrics and cause double-billing.`
	warningDeprecatedImplicitActiveGate                      = `DynaKube's specification enables extensions without an ActiveGate explicitly set. Extensions requires an ActiveGate, and the Operator will implicitly create one, but this behavior has been deprecetad.`
)

func extensionControllerImage(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	log := logd.FromContext(ctx)

	if !dk.Extensions().IsAnyEnabled() {
		return ""
	}

	if dk.FF().IsPublicRegistry() {
		return ""
	}

	if !dk.Spec.Templates.ExtensionExecutionController.ImageRef.HasImage() {
		log.Info("requested dynakube doesn't specify the ExtensionExecutionController image.")

		return errorExtensionExecutionControllerImageNotSpecified
	}

	return ""
}

func conflictingAPIURLForExtensions(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	log := logd.FromContext(ctx)

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

func extensionControllerPVCStorageDevice(ctx context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	log := logd.FromContext(ctx)

	if !dk.Extensions().IsAnyEnabled() {
		return ""
	}

	if extensionControllerMutuallyExclusivePVCSettings(dk) {
		log.Info("requested dynakube specifies mutually exclusive VolumeClaimTemplate settings for ExtensionExecutionController.")

		return errorExtensionExecutionControllerInvalidPVCConfiguration
	}

	return ""
}

func extensionControllerMutuallyExclusivePVCSettings(dk *dynakube.DynaKube) bool {
	return ptr.Deref(dk.Spec.Templates.ExtensionExecutionController.UseEphemeralVolume, false) && dk.Spec.Templates.ExtensionExecutionController.PersistentVolumeClaim != nil
}

func deprecatedImplicitActiveGate(ctx context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	log := logd.FromContext(ctx)

	if dk.Extensions().IsAnyEnabled() && dk.Spec.ActiveGate == nil {
		log.Info("requested dynakube configures extensions without an explicit activeGate section.")

		return warningDeprecatedImplicitActiveGate
	}

	return ""
}
