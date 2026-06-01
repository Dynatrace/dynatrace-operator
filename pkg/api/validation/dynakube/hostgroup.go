package validation

import (
	"context"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
)

const (
	errorInvalidHostGroupProperty = "The DynaKube's specification has an invalid Host Group value set using the oneAgent.hostGroup property. Make sure to remove all whitespace characters (newline, tab, carriage return, null) from the Host Group value in your custom resource."
	errorInvalidHostGroupAsParam  = "The DynaKube's specification has an invalid Host Group value set using the --set-host-group argument. Make sure to remove all whitespace characters (newline, tab, carriage return, null) from the Host Group value in your custom resource."
)

func invalidOneAgentHostGroup(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if strings.ContainsFunc(dk.OneAgent().HostGroup, isWhiteSpaceCharacter) {
		return errorInvalidHostGroupProperty
	}

	if strings.ContainsFunc(dk.OneAgent().GetHostGroupAsParam(), isWhiteSpaceCharacter) {
		return errorInvalidHostGroupAsParam
	}

	return ""
}
