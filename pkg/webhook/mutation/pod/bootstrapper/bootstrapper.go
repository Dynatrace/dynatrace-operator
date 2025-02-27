package bootstrapper

import (
	"context"

	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
)

func HandleBootstrapperMutation(ctx context.Context, mutationRequest *dtwebhook.MutationRequest, mutators []dtwebhook.PodMutator) (bool, error) {
	isMutated := false

	for _, mutator := range mutators {
		if !mutator.Enabled(mutationRequest.BaseRequest) {
			continue
		}

		if err := mutator.Mutate(ctx, mutationRequest); err != nil {
			return false, err
		}

		isMutated = true
	}

	if !isMutated {
		log.Info("no mutation is enabled")
	}

	return isMutated, nil
}

func HandleBootstrapperReinvocation(reinvocationRequest *dtwebhook.ReinvocationRequest, mutators []dtwebhook.PodMutator) bool {
	needsUpdate := false

	for _, mutator := range mutators {
		if mutator.Enabled(reinvocationRequest.BaseRequest) {
			if update := mutator.Reinvoke(reinvocationRequest); update {
				needsUpdate = true
			}
		}
	}

	return needsUpdate
}
