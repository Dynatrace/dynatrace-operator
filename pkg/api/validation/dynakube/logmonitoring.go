package validation

import (
	"context"
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errorConflictingLogMonitoring = "The DynaKube's specification tries to enable LogMonitoring in a namespace where another DynaKube already deploys the OneAgent, which is not supported. The conflicting DynaKubes: %s"

	errorConflictingOneAgentSpec = "The DynaKube's specification tries to enable LogMonitoring and OneAgent at the same time, which is not supported. Please disable the LogMonitoring or OneAgent"
)

func conflictingLogMonitoringNodeSelector(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	if !dk.LogMonitoring().IsEnabled() {
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
			if hasConflictingMatchLabels(dk.LogMonitoring().NodeSelector, item.OneAgentNodeSelector()) {
				log.Info("requested dynakube has conflicting LogMonitoring nodeSelector", "name", dk.Name, "namespace", dk.Namespace)
				conflictingDynakubes = append(conflictingDynakubes, item.Name)
			}
		}
	}

	if len(conflictingDynakubes) > 0 {
		return fmt.Sprintf(errorConflictingLogMonitoring, strings.Join(conflictingDynakubes, ", "))
	}

	return ""
}
