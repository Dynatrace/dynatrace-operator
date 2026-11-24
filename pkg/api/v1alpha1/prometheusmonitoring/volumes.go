// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package prometheusmonitoring

import "github.com/Dynatrace/dynatrace-operator/pkg/consts"

const (
	ConfigVolumeName = "dynatrace-otel-config"
	ConfigMountDir   = "/conf"

	CACertsVolumeName  = "cacerts"
	TrustedCAMountPath = "/tls/custom/cacerts"
	TokenVolumeName    = "dt-token"
	TokenMountPath     = consts.DTComponentsSecretsRootDir + "/tokens"
)

var (
	GatewayVolumeNames      = []string{ConfigVolumeName, TokenVolumeName, CACertsVolumeName}
	GatewayVolumeMountPaths = []string{ConfigMountDir, TokenMountPath, TrustedCAMountPath}

	ScraperVolumeNames      = []string{ConfigVolumeName}
	ScraperVolumeMountPaths = []string{ConfigMountDir}

	SelfMonitoringVolumeNames      = []string{ConfigVolumeName, TokenVolumeName, CACertsVolumeName}
	SelfMonitoringVolumeMountPaths = []string{ConfigMountDir, TokenMountPath, TrustedCAMountPath}

	TargetAllocatorVolumeNames      = []string{ConfigVolumeName}
	TargetAllocatorVolumeMountPaths = []string{ConfigMountDir}
)
