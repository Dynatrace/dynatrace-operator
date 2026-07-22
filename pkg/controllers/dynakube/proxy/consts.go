// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package proxy

import "github.com/Dynatrace/dynatrace-operator/pkg/consts"

const (
	hostField     = "host"
	portField     = "port"
	usernameField = "username"
	passwordField = "password"
	schemeField   = "scheme"

	SecretMountPath  = consts.DTComponentsSecretsRootDir + "/internal-proxy"
	SecretVolumeName = "internal-proxy-secret-volume"
)
