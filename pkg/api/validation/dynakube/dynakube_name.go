package validation

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	errorNoDNS1053Label = `The DynaKube name violates DNS-1035, ` +
		// This error message is copied from the apimachinery validation code.
		`a DNS-1035 label must consist of lower case alphanumeric characters or '-', start with an alphabetic character, and end with an alphanumeric character (e.g. 'my-name',  or 'abc-123', regex used for validation is '[a-z]([-a-z0-9]*[a-z0-9])?')`

	errorNameTooLong = `The length limit for the name of a DynaKube is %d, because it is the base for the name of resources related to the DynaKube. (example: dkName-activegate-<some-hash>).
    The limit is necessary because kubernetes uses the name of some resources (example: StatefulSet) for the label value, which has a limit of 63 characters. (see https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set)`
)

func nameInvalid(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if dk.Name == "" {
		// Make unit testing easier. This can never happen in an actual cluster.
		return ""
	}

	// Always call this before DNS1035 validation to prevent false positives due to too long name.
	if err := nameTooLong(dk); err != "" {
		return err
	}

	return nameViolatesDNS1035(dk)
}

func nameViolatesDNS1035(dk *dynakube.DynaKube) string {
	errs := validation.IsDNS1035Label(dk.Name)
	if len(errs) == 0 {
		return ""
	}

	return errorNoDNS1053Label
}

func nameTooLong(dk *dynakube.DynaKube) string {
	nameLen := len(dk.Name)
	maxLength := dynakube.MaxNameLength

	if nameLen > maxLength {
		return fmt.Sprintf(errorNameTooLong, maxLength)
	}

	return ""
}
