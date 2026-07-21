package validation

import (
	"context"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/sanitize"
)

const (
	errorInvalidNetworkZone = "The DynaKube's specification has an invalid Network Zone value set. Make sure to remove forbidden characters (newline, tab, carriage return, null) from the Network Zone value in your custom resource."
)

func invalidNetworkZone(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if strings.ContainsAny(dk.Spec.NetworkZone, sanitize.InvalidCommandLineCharset) {
		return errorInvalidNetworkZone
	}

	return ""
}
