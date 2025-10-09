package otlp

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

var (
	log = logd.Get().WithName("pod-mutation-otlp")
)
