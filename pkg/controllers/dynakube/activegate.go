package dynakube

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/pkg/errors"
)

func (controller *Controller) reconcileActiveGate(ctx context.Context, dk *dynakube.DynaKube, dtc dynatrace.Client) error {
	reconciler := controller.activeGateReconcilerBuilder(controller.client, controller.apiReader, dk, dtc, controller.tokens)

	err := reconciler.Reconcile(ctx)
	if err != nil {
		return errors.WithMessage(err, "failed to reconcile ActiveGate")
	}

	return nil
}
