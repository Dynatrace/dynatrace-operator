package service

import "github.com/Dynatrace/dynatrace-operator/pkg/logd"

var (
	log = logd.Get().WithName("otelc-service")
)
