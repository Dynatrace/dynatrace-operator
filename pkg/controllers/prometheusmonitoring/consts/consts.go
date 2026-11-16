// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package consts

import "github.com/Dynatrace/dynatrace-operator/pkg/consts"

const (
	ConfigVolumeName = "opentelemetry-collector-configmap"
	ConfigMountDir   = "/conf"

	CACertsVolumeName               = "cacerts"
	TrustedCAMountPath              = "/tls/custom/cacerts"
	TokenVolumeName                 = "dt-token"
	TokenMountPath                  = consts.DTComponentsSecretsRootDir + "/tokens"
	TargetAllocatorConfigVolumeName = "config"
)

var (
	GatewayVolumeNames      = []string{ConfigVolumeName, TokenVolumeName, CACertsVolumeName}
	GatewayVolumeMountPaths = []string{ConfigMountDir, TokenMountPath, TrustedCAMountPath}

	ScraperVolumeNames      = []string{ConfigVolumeName}
	ScraperVolumeMountPaths = []string{ConfigMountDir}

	TargetAllocatorVolumeNames      = []string{TargetAllocatorConfigVolumeName}
	TargetAllocatorVolumeMountPaths = []string{ConfigMountDir}
)
