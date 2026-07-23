// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package tenant

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	dtsettings "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
)

func BuildSettingsClient(secretConfig Secret) (dtsettings.Client, error) {
	dtClient, err := dynatrace.NewClient(
		dynatrace.WithBaseURL(secretConfig.APIURL),
		dynatrace.WithAPIToken(secretConfig.TokensWithSettingsScope().APIToken),
		dynatrace.WithSkipCertificateValidation(false))
	if err != nil {
		return nil, err
	}

	return dtClient.Settings, nil
}
