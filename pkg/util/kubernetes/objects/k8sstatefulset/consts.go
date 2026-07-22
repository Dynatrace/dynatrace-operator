// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8sstatefulset

import "github.com/Dynatrace/dynatrace-operator/pkg/api"

const (
	AnnotationPVCHash = api.InternalFlagPrefix + "pvc-hash"
)
