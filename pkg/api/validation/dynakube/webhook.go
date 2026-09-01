// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	latest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	v1beta5 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/validation"
	ctrl "sigs.k8s.io/controller-runtime"
)

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	validator := New(mgr.GetAPIReader())

	if err := validation.SetupWebhookForType(mgr, &v1beta5.DynaKube{}, validator); err != nil {
		return err
	}

	return validation.SetupWebhookForType(mgr, &latest.DynaKube{}, validator)
}
