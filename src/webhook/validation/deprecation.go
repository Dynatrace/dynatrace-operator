package validation

import (
	"fmt"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
)

const (
	featureDeprecatedWarningMessage = `DEPRECATED: %s`
)

func deprecatedFeatureFlagFormat(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.Annotations == nil {
		return ""
	}

	deprecatedPrefix := dynatracev1beta1.DeprecatedFeatureFlagPrefix
	if len(dynatracev1beta1.FlagsWithPrefix(dynakube, deprecatedPrefix)) > 0 {
		return fmt.Sprintf(featureDeprecatedWarningMessage, "'alpha.operator.dynatrace.com/feature-' prefix will be replaced with the 'feature.dynatrace.com/' prefix for dynakube feature-flags")
	}

	return ""
}
