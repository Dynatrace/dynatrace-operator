package conversion

import (
	"encoding/json"
)

const (
	Prefix = "conversion.internal.dynatrace.com/"

	AutoUpdateKey = Prefix + "auto-update"
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
	AutoUpdate Field[bool]
}

func NewRemovedFields(annotations map[string]string) *RemovedFields {
	return &RemovedFields{
		AutoUpdate: Field[bool]{name: AutoUpdateKey, data: annotations},
	}
}
