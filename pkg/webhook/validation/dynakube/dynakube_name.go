package dynakube

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	errorNoDNS1053Label = `The DynaKube's specification violates DNS-1035.
    [a DNS-1035 label must consist of lower case alphanumeric characters or '-', start with an alphabetic character, and end with an alphanumeric character (e.g. 'my-name',  or 'abc-123', regex used for validation is '[a-z]([-a-z0-9]*[a-z0-9])?')]
	`
)

func nameViolatesDNS1035(_ context.Context, dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
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
