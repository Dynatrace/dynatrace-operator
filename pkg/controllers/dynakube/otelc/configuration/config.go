package configuration

import "github.com/Dynatrace/dynatrace-operator/pkg/logd"

var (
	log = logd.Get().WithName("otelc-config")
)
