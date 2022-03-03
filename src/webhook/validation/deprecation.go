package validation

import (
	"fmt"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
)

const (
	featureDeprecatedWarningMessage = `DEPRACETED: %s`
)

func deprecatedFeatureFlagFormat(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.Annotations == nil {
		return ""
	}
	for annotation := range dynakube.Annotations {
		if strings.HasPrefix(annotation, dynatracev1beta1.DeprecatedFeatureFlagPrefix+dynatracev1beta1.FeatureFlagAnnotationPrefix) {
			return fmt.Sprintf(featureDeprecatedWarningMessage, "feature flags with 'alpha.' prefix")
		}
	}
	return ""
}
