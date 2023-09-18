package dynakube

import (
	"unicode"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
)

const (
	errorDigitInName = `The DynaKube's specification has an invalid name: It starts with a digit.
	`
)

func nameStartsWithDigit(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	dynakubeName := dynakube.Name
	var firstChar rune
	if dynakubeName != "" {
		firstChar = rune(dynakubeName[0])
	}

	if unicode.IsDigit(firstChar) {
		return errorDigitInName
	}

	return ""
}
