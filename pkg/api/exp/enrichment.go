// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package exp

const (
	EnrichmentEnableAttributesDTKubernetes = FFPrefix + "enable-attributes-dt.kubernetes"
)

func (ff *FeatureFlags) EnableAttributesDTKubernetes() bool {
	defaultVal := !ff.hasPlatformToken

	return ff.getBoolWithDefault(EnrichmentEnableAttributesDTKubernetes, defaultVal)
}
