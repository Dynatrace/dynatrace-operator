package validation

import (
	"context"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
)

const (
	errorInvalidNetworkZone = "The DynaKube's specification has an invalid Network Zone value set. Make sure to remove all whitespace characters (newline, tab, carriage return, null) from the Network Zone value in your custom resource."
)

func invalidNetworkZone(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if strings.ContainsFunc(dk.Spec.NetworkZone, isWhiteSpaceCharacter) {
		return errorInvalidNetworkZone
	}

	return ""
}

func isWhiteSpaceCharacter(r rune) bool {
	switch r {
	case '\n', '\t', '\r', '\x00':
		return true
	}

	return false
}
