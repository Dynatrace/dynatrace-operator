// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	v1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	v1alpha2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/validation"
	ctrl "sigs.k8s.io/controller-runtime"
)

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	validator := New(mgr.GetAPIReader(), mgr.GetConfig())

	if err := validation.SetupWebhookForType(mgr, &v1alpha1.EdgeConnect{}, validator); err != nil {
		return err
	}

	return validation.SetupWebhookForType(mgr, &v1alpha2.EdgeConnect{}, validator)
}
