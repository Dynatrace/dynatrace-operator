package dynakube

import (
	"context"
	"fmt"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
)

const (
	featurePreviewWarningMessage = `%s feature is in PREVIEW.`
	basePreviewWarning           = "PREVIEW features are NOT production ready and you may run into bugs."
)

func syntheticPreviewWarning(_ context.Context, dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.IsSyntheticMonitoringEnabled() {
		log.Info(fmt.Sprintf("DynaKube with %s was applied, warning was provided.", capability.SyntheticName))
		return fmt.Sprintf(featurePreviewWarningMessage, capability.SyntheticName)
	}

	return ""
}
