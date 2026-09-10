// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/dtprometheus"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func setupWebhookWithManager(mgr ctrl.Manager, obj runtime.Object, validator admission.Validator[runtime.Object]) error {
	return ctrl.NewWebhookManagedBy(mgr, obj).WithCustomValidator(validator).Complete() //nolint
}

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	validator := New(mgr.GetAPIReader())

	return setupWebhookWithManager(mgr, &dtprometheus.DTPrometheus{}, validator)
}
