// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package conversion

import (
	"encoding/json"
)

const (
	Prefix = "conversion.internal.dynatrace.com/"

	AutoUpdateKey          = Prefix + "auto-update"
	DefaultOTelColImageKey = Prefix + "default-otelc-image"

	NetworkZoneFallbackModeKey = Prefix + "network-zone-fallback-mode"
)

type Field[T any] struct {
	data map[string]string
	name string
}

func (f Field[T]) Get() *T {
	raw, exists := f.data[f.name]
	if !exists {
		return nil
	}

	var value T
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return nil
	}

	return &value
}

func (f Field[T]) Set(value *T) {
	if value == nil {
		delete(f.data, f.name)

		return
	}

	raw, _ := json.Marshal(*value)
	f.data[f.name] = string(raw)
}

// RemovedFields are fields that existed in the older versions, but were removed from the latest (hub) version.
// While converting to the latest version they are stored in the annotations of the latest object,
// so they are not lost, and restored when converting back to the older version.
type RemovedFields struct {
	AutoUpdate          Field[bool]
	DefaultOTelColImage Field[bool]
}

func NewRemovedFields(annotations map[string]string) *RemovedFields {
	return &RemovedFields{
		AutoUpdate:          Field[bool]{name: AutoUpdateKey, data: annotations},
		DefaultOTelColImage: Field[bool]{name: DefaultOTelColImageKey, data: annotations},
	}
}

// Cleanup removes the annotations of the RemovedFields,
// it is meant to be used when converting to an older version, where the fields exist.
func (rf *RemovedFields) Cleanup() {
	rf.AutoUpdate.Set(nil)
	rf.DefaultOTelColImage.Set(nil)
}

// AddedFields are fields that exist in the latest (hub) version, but not in the older versions.
// While converting to an older version they are stored in the annotations of the older object,
// so they are not lost, and restored when converting back to the latest version.
type AddedFields struct {
	NetworkZoneFallbackMode Field[string]
}

func NewAddedFields(annotations map[string]string) *AddedFields {
	return &AddedFields{
		NetworkZoneFallbackMode: Field[string]{name: NetworkZoneFallbackModeKey, data: annotations},
	}
}

// Cleanup removes the annotations of the AddedFields,
// it is meant to be used when converting to the latest version, where the fields exist.
func (af *AddedFields) Cleanup() {
	af.NetworkZoneFallbackMode.Set(nil)
}
