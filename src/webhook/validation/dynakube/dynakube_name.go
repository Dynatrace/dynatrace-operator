package dynakube

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	errorDigitInName = `The DynaKube's specification has an invalid name: It starts with a digit.
	`
)

func nameStartsWithDigit(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	dynakubeName := dynakube.Name
	var errs []string
	if dynakubeName != "" {
		errs = validation.IsDNS1035Label(dynakubeName)
	}

	if len(errs) == 0 {
		return ""
	}
	return errorDigitInName
}
