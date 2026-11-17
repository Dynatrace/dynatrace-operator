// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package conversion

import (
	"encoding/json"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
)

const (
	Prefix = "conversion.internal.dynatrace.com/"

	AutoUpdateKey          = Prefix + "auto-update"
	DefaultOTelColImageKey = Prefix + "default-otelc-image"
	TLSModeKey             = Prefix + "tls-mode"
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

type RemovedFields struct {
	AutoUpdate          Field[bool]
	DefaultOTelColImage Field[bool]
}

func NewRemovedFields(annotations map[string]string) *RemovedFields {
	// We don't want to initialize dk.Annotation map here
	return &RemovedFields{
		AutoUpdate:          Field[bool]{name: AutoUpdateKey, data: annotations},
		DefaultOTelColImage: Field[bool]{name: DefaultOTelColImageKey, data: annotations},
	}
}

func CleanupAnnotations(annotations map[string]string) {
	delete(annotations, AutoUpdateKey)
	delete(annotations, DefaultOTelColImageKey)
}

type AddedFields struct {
	// TLSMode added to v1beta6
	TLS Field[dynakube.TLSSpec]
}

func NewAddedFields(annotations map[string]string) *AddedFields {
	// We don't want to initialize dk.Annotation map here
	return &AddedFields{
		TLS: Field[dynakube.TLSSpec]{name: TLSModeKey, data: annotations},
	}
}

func CleanupAnnotationsForAddedFields(annotations map[string]string) {
	delete(annotations, TLSModeKey)
}
