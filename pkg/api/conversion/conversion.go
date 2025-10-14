package conversion

import (
	"strconv"

	"k8s.io/utils/ptr"
)

const (
	Prefix = "conversion.internal.dynatrace.com/"

	AutoUpdateKey = Prefix + "auto-update"
)

type RemovedFields struct {
	annotations map[string]string
}

func NewRemovedFields(annotations map[string]string) *RemovedFields {
	return &RemovedFields{annotations: annotations}
}

func (rf *RemovedFields) GetAutoUpdate() *bool {
	return rf.getBool(AutoUpdateKey)
}

func (rf *RemovedFields) SetAutoUpdate(autoUpdate *bool) {
	rf.setBool(AutoUpdateKey, autoUpdate)
}

func (rf *RemovedFields) getBool(key string) *bool {
	if value, exists := rf.annotations[key]; exists {
		b, err := strconv.ParseBool(value)
		if err == nil {
			return ptr.To(b)
		}
	}

	return nil
}

func (rf *RemovedFields) setBool(key string, value *bool) {
	if value == nil {
		delete(rf.annotations, key)

		return
	}

	rf.annotations[key] = strconv.FormatBool(*value)
}
