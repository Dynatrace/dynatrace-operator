// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/dtprometheus"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/validation"
	ctrl "sigs.k8s.io/controller-runtime"
)

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	validator := New(mgr.GetAPIReader())

	return validation.SetupWebhookForType(mgr, &dtprometheus.DTPrometheus{}, validator)
}
