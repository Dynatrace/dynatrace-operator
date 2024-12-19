package otelc

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

var (
	log = logd.Get().WithName("otel-collector")
)
