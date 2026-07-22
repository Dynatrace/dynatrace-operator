// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kubemon"
)

func (dk *DynaKube) KubernetesMonitoring() *kubemon.KubeMon {
	km := &kubemon.KubeMon{
		Spec:   dk.Spec.KubernetesMonitoring,
		Status: &dk.Status.KubernetesMonitoring,
	}
	km.SetName(dk.Name)

	return km
}
