// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dynakube

func (dk *DynaKube) MetadataEnrichmentEnabled() bool {
	return dk.Spec.MetadataEnrichment.Enabled != nil && *dk.Spec.MetadataEnrichment.Enabled
}
