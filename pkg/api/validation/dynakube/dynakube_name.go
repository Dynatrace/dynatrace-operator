package validation

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	errorNoDNS1053Label = `The DynaKube's specification violates DNS-1035.
    [a DNS-1035 label must consist of lower case alphanumeric characters or '-', start with an alphabetic character, and end with an alphanumeric character (e.g. 'my-name',  or 'abc-123', regex used for validation is '[a-z]([-a-z0-9]*[a-z0-9])?')]
	`

	errorNameTooLong = `The length limit for the name of a DynaKube is %d, because it is the base for the name of resources related to the DynaKube. (example: dkName-activegate-<some-hash>).
    The limit is necessary because kubernetes uses the name of some resources (example: StatefulSet) for the label value, which has a limit of 63 characters. (see https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set)`

	sqlExecutorTooLongSuffix = `.
    When using SQL extension executors, the Deployment name format requires the DynaKube name to be shorter than usual.`
)

func nameViolatesDNS1035(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if dk.Name == "" {
		// Make unit testing easier. This can never happen in an actual cluster.
		return ""
	}

	errs := validation.IsDNS1035Label(dk.Name)

	if len(errs) == 0 {
		return ""
	}

	return errorNoDNS1053Label
}

func nameTooLong(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if dk.Name == "" {
		// Make unit testing easier. This can never happen in an actual cluster.
		return ""
	}

	nameLen := len(dk.Name)
	maxLength := dynakube.MaxNameLength

	var suffix string

	if dk.Extensions().IsDatabasesEnabled() {
		// Extensions make use Deployments which have different name length requirements than StatefulSet and DaemonSets.
		maxLength = maxNameLengthForSQLExecutor(dk)
		if maxLength < dynakube.MaxNameLength {
			suffix = sqlExecutorTooLongSuffix
		}
	}

	if nameLen > maxLength {
		return fmt.Sprintf(errorNameTooLong, maxLength) + suffix
	}

	return ""
}

func maxNameLengthForSQLExecutor(dk *dynakube.DynaKube) int {
	// Pod names are limited to 63 characters and Kubernetes will cut off characters from the owner resource name to ensure that they can be deployed.
	// Max length for a Deployment to ensure that nothing gets cut off is: 57 (+5 for random suffix +1 for separating hyphen)
	const (
		maxDeploymentNameLength = 57
		infixLen                = len(extensions.SQLExecutorInfix)
	)

	var maxIDLen int
	for _, db := range dk.Extensions().Databases {
		maxIDLen = max(maxIDLen, len(db.ID))
	}

	return min(dynakube.MaxNameLength, maxDeploymentNameLength-infixLen-maxIDLen)
}
