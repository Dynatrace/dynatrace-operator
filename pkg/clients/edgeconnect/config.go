package edgeconnect

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
)

var (
	log = logger.Factory.GetLogger("edgeConnectClient")
)

var DefaultOauthScopes = []string{"test"}

const DefaultTokenURL = "https://sso-dev.dynatracelabs.com/sso/oauth2/token"
