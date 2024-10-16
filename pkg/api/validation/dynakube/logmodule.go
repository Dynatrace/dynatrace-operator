package validation

import (
	"context"
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errorConflictingLogModule = "The DynaKube's specification tries to enable LogModule in a namespace where another DynaKube already deploys the OneAgent, which is not supported. The conflicting DynaKubes: %s"

	errorConflictingOneAgentSpec = "The DynaKube's specification tries to enable LogModule and OneAgent at the same time, which is not supported. Please disable the LogModule or OneAgent"
)

func conflictingLogModuleNodeSelector(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	if !dk.Spec.LogModule.Enabled {
		return ""
	}

	if dk.NeedsOneAgent() {
		return errorConflictingOneAgentSpec
	}

	validDynakubes := &dynakube.DynaKubeList{}
	if err := dv.apiReader.List(ctx, validDynakubes, &client.ListOptions{Namespace: dk.Namespace}); err != nil {
		log.Info("error occurred while listing dynakubes", "err", err.Error())

		return ""
	}

	conflictingDynakubes := []string{}

	for _, item := range validDynakubes.Items {
		if item.Name == dk.Name {
			continue
		}

		if item.NeedsOneAgent() {
			if hasConflictingMatchLabels(dk.LogModuleNodeSelector(), item.OneAgentNodeSelector()) {
				log.Info("requested dynakube has conflicting LogModule nodeSelector", "name", dk.Name, "namespace", dk.Namespace)
				conflictingDynakubes = append(conflictingDynakubes, item.Name)
			}
		}
	}

	if len(conflictingDynakubes) > 0 {
		return fmt.Sprintf(errorConflictingLogModule, strings.Join(conflictingDynakubes, ", "))
	}

	return ""
}
