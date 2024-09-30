package validation

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
)

const (
	errorNameTooLong = `The length limit for the name of a EdgeConnect is %d, because it is the base for the name of resources related to the EdgeConnect.
	The limit is necessary because kubernetes uses the name of some resources for the label value, which has a limit of 63 characters. (see https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set)`
)

func nameTooLong(_ context.Context, _ *Validator, ec *edgeconnect.EdgeConnect) string {
	edgeConnectName := ec.Name
	if edgeConnectName != "" && len(edgeConnectName) > edgeconnect.MaxNameLength {
		return fmt.Sprintf(errorNameTooLong, edgeconnect.MaxNameLength)
	}

	return ""
}
