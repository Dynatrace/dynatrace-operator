package conversion

import (
	"encoding/json"
)

const (
	Prefix = "conversion.internal.dynatrace.com/"

	AutoUpdateKey        = Prefix + "auto-update"
	DefaultOTELCImageKey = Prefix + "default-otelc-image"
	OAMaxUnavailableKey  = Prefix + "oneagent-max-unavailable"
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
	AutoUpdate        Field[bool]
	DefaultOTELCImage Field[bool]
	OAMaxUnavailable  Field[int]
}

func NewRemovedFields(annotations map[string]string) *RemovedFields {
	return &RemovedFields{
		AutoUpdate:        Field[bool]{name: AutoUpdateKey, data: annotations},
		DefaultOTELCImage: Field[bool]{name: DefaultOTELCImageKey, data: annotations},
		OAMaxUnavailable:  Field[int]{name: OAMaxUnavailableKey, data: annotations},
	}
}

func CleanupAnnotations(annotations map[string]string) {
	delete(annotations, AutoUpdateKey)
	delete(annotations, DefaultOTELCImageKey)
	delete(annotations, OAMaxUnavailableKey)
}
