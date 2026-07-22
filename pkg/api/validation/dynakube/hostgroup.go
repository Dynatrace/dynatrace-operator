// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/sanitize"
)

const (
	errorInvalidHostGroupProperty = "The DynaKube's specification has an invalid Host Group value set using the oneAgent.hostGroup property. Make sure to remove forbidden characters (newline, tab, carriage return, null) from the Host Group value in your custom resource."
	errorInvalidHostGroupAsParam  = "The DynaKube's specification has an invalid Host Group value set using the --set-host-group argument. Make sure to remove forbidden characters (newline, tab, carriage return, null) from the Host Group value in your custom resource."
)

func invalidOneAgentHostGroup(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if strings.ContainsAny(dk.OneAgent().HostGroup, sanitize.InvalidCommandLineCharset) {
		return errorInvalidHostGroupProperty
	}

	return ""
}
