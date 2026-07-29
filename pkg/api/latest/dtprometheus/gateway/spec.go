// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package gateway

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dtprometheus/common"
	appsv1 "k8s.io/api/apps/v1"
)

// Gateway wraps the gateway Spec together with the owning DtPrometheus name so
// derived state (such as Kubernetes resource names) can be computed.
type Gateway struct {
	*Spec

	name string
}

// +kubebuilder:object:generate=true

// Spec configures the gateway pool StatefulSet (tier 2): a StatefulSet of OTel
// Collectors that run stateful processors and export to Dynatrace via OTLP/HTTP.
type Spec struct {
	common.Spec `json:",inline"`

	// StatefulSet update strategy for the gateway pool. partition: N means only
	// pods with ordinal >= N are updated; set a non-zero value for canary rollouts.
	// +kubebuilder:validation:Optional
	UpdateStrategy appsv1.StatefulSetUpdateStrategy `json:"updateStrategy,omitzero"`
}
