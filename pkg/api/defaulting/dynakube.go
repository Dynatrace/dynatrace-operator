// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package defaulting

import (
	"cmp"
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/system"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const networkZonePrefix = "k8s-"

type dkDefaulter struct {
	apiReader client.Reader
}

var _ admission.Defaulter[*dynakube.DynaKube] = &dkDefaulter{}

// Default applies all defaulting rules to the DynaKube.
func (d *dkDefaulter) Default(ctx context.Context, dk *dynakube.DynaKube) error {
	for _, rule := range d.rules() {
		if err := rule(ctx, dk); err != nil {
			return err
		}
	}

	return nil
}

func (d *dkDefaulter) rules() []func(context.Context, *dynakube.DynaKube) error {
	return []func(context.Context, *dynakube.DynaKube) error{
		d.defaultNetworkZone,
		// next rule
	}
}

// defaultNetworkZone defaults spec.networkZone to "k8s-<cluster uid>" if it is not set,
// and in that case also defaults spec.networkZoneFallbackMode.
func (d *dkDefaulter) defaultNetworkZone(ctx context.Context, dk *dynakube.DynaKube) error {
	_, log := logd.NewFromContext(ctx, "dynakube-defaulting")

	if dk.Spec.NetworkZone != "" {
		return nil
	}

	uid, err := system.GetUID(ctx, d.apiReader)
	if err != nil {
		return fmt.Errorf("failed to determine cluster UID for network zone defaulting: %w", err)
	}

	dk.Spec.NetworkZone = networkZonePrefix + string(uid)
	dk.Spec.NetworkZoneFallbackMode = cmp.Or(dk.Spec.NetworkZoneFallbackMode, dynakube.NZFallbackModeDefault)

	log.Info("defaulted network zone", "networkZone", dk.Spec.NetworkZone, "fallbackMode", dk.Spec.NetworkZoneFallbackMode)

	return nil
}
