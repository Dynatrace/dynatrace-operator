package configsecret

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

const (
	DeploymentConfigFilename = "deployment.conf"
)

var (
	log = logd.Get().WithName("logmodule-config-secret")
)
