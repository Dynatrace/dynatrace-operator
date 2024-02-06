package dynakube

import (
	"context"
	"fmt"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	errorNoDNS1053Label = `The DynaKube's specification violates DNS-1035.
    [a DNS-1035 label must consist of lower case alphanumeric characters or '-', start with an alphabetic character, and end with an alphanumeric character (e.g. 'my-name',  or 'abc-123', regex used for validation is '[a-z]([-a-z0-9]*[a-z0-9])?')]
	`

	errorNameTooLong = `The length limit for the name of a DynaKube is %d, because it is the base for the name of resources related to the DynaKube. (example: dkName-activegate-<some-hash>)
	The limit is necessary because kubernetes uses the name of some resources (example: StatefulSet) for the label value, which has a limit of 63 characters. (see https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set)`
)

func nameViolatesDNS1035(_ context.Context, _ *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	dynakubeName := dynakube.Name

	var errs []string

	if dynakubeName != "" {
		errs = validation.IsDNS1035Label(dynakubeName)
	}

	if len(errs) == 0 {
		return ""
	}

	return errorNoDNS1053Label
}

func nameTooLong(_ context.Context, _ *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	dynakubeName := dynakube.Name
	if dynakubeName != "" && len(dynakubeName) > dynatracev1beta1.MaxNameLength {
		return fmt.Sprintf(errorNameTooLong, dynatracev1beta1.MaxNameLength)
	}

	return ""
}
