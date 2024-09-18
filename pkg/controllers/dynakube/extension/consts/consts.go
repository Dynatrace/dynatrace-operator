package consts

import "time"

const (
	// secret
	EecTokenSecretKey         = "eec.token"
	EecTokenSecretValuePrefix = "EEC dt0x01"

	OtelcTokenSecretKey         = "otelc.token"
	OtelcTokenSecretValuePrefix = "dt0x01"

	SecretSuffix = "-extensions-token"

	// shared volume name between eec and OtelC
	ExtensionsTokensVolumeName = "tokens"

	ExtensionsSecretConditionType  = "ExtensionsSecret"
	ExtensionsServiceConditionType = "ExtensionsService"

	ExtensionsControllerSuffix        = "-extensions-controller"
	ExtensionsCollectorComPort        = 14599
	ExtensionsCollectorTargetPortName = "collector-com"

	ExtensionsSelfSignedTlsSecretSuffix     = "-extensions-controller-tls"
	ExtensionsSelfSignedTlsCommonNameSuffix = "-extensions-controller.dynatrace"
	ExtensionsSelfSignedTlsRenewalThreshold = 12 * time.Hour

	// TlsKeyDataName is the key used to store a TLS private key in the secret's data field.
	TlsKeyDataName = "tls.key"

	// TlsCrtDataName is the key used to store a TLS certificate in the secret's data field.
	TlsCrtDataName = "tls.crt"
)
