// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package crdstoragemigration

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// defaultingTrigger describes a defaulting rule (implemented by a defaulting webhook) that has to be applied
// to already existing DynaKubes during an operator upgrade.
// Defaulting webhooks only run on CREATE/UPDATE, so existing DynaKubes are "updated" without changes to trigger them.
//
// To trigger a new defaulting rule, add a defaultingTrigger to defaultingTriggers.
type defaultingTrigger struct {
	// name is only used for logging.
	name string

	// needsDefaulting reports whether the given DynaKube is missing values the defaulting webhook would set.
	needsDefaulting func(dk *dynakube.DynaKube) bool
}

var defaultingTriggers = []defaultingTrigger{
	{
		name: "network zone",
		needsDefaulting: func(dk *dynakube.DynaKube) bool {
			return dk.Spec.NetworkZone == ""
		},
	},
}

// TriggerDefaulting applies the defaultingTriggers to the existing DynaKubes.
//
// Every DynaKube that needs defaulting according to any of the defaultingTriggers is "updated".
// A DynaKube is updated at most once, no matter how many triggers apply to it.
// The DynaKubes are read using the latest API version, which is the one the defaulting webhooks are registered for.
//
// It must only be called from the operator's init container, after the webhook is ready, and not from the
// CRD storage migration job: the job runs during the Helm install/upgrade with the old CRD and without the new
// defaulting webhook, so the DynaKubes couldn't be defaulted at that point.
func TriggerDefaulting(ctx context.Context, clt client.Client, namespace string) error {
	log := logd.FromContext(ctx)

	var dynakubeList dynakube.DynaKubeList

	err := clt.List(ctx, &dynakubeList, client.InNamespace(namespace))
	if err != nil {
		return errors.Wrap(err, "failed to list DynaKube instances")
	}

	for i := range dynakubeList.Items {
		dk := &dynakubeList.Items[i]

		var reasons []string

		for _, trigger := range defaultingTriggers {
			if trigger.needsDefaulting(dk) {
				reasons = append(reasons, trigger.name)
			}
		}

		if len(reasons) == 0 {
			continue
		}

		log.Info("updating DynaKube to trigger defaulting", "name", dk.Name, "namespace", dk.Namespace, "triggers", reasons)

		err = clt.Update(ctx, dk)
		if err != nil {
			return errors.Wrapf(err, "failed to update DynaKube %s/%s", dk.Namespace, dk.Name)
		}
	}

	return nil
}
