package dynakube

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/pkg/errors"
)

func (controller *Controller) reconcileActiveGate(ctx context.Context, dk *dynakube.DynaKube, dtc *dynatrace.Client) error {
	err := controller.activeGateReconciler.Reconcile(ctx, dk, dtc, controller.tokens)
	if err != nil {
		return errors.WithMessage(err, "failed to reconcile ActiveGate")
	}

	return nil
}
