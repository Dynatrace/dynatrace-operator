package consts

import "github.com/Dynatrace/dynatrace-operator/pkg/api"

const (
	ExtensionsAnnotationSecretHash = api.InternalFlagPrefix + "secret-hash"

	// secret
	EecTokenSecretKey         = "eec.token"
	EecTokenSecretValuePrefix = "EEC dt0x01"

	OtelcTokenSecretKey         = "otelc.token"
	OtelcTokenSecretValuePrefix = "dt0x01"

	// shared volume name between eec and OtelC
	ExtensionsTokensVolumeName = "tokens"

	ExtensionsSecretConditionType  = "ExtensionsSecret"
	ExtensionsServiceConditionType = "ExtensionsService"

	ExtensionsControllerSuffix        = "-extensions-controller"
	ExtensionsCollectorComPort        = 14599
	ExtensionsCollectorTargetPortName = "collector-com"

	ExtensionsSelfSignedTLSSecretSuffix     = "-extensions-controller-tls"
	ExtensionsSelfSignedTLSCommonNameSuffix = "-extensions-controller.dynatrace"

	// TLSKeyDataName is the key used to store a TLS private key in the secret's data field.
	TLSKeyDataName = "tls.key"

	// TLSCrtDataName is the key used to store a TLS certificate in the secret's data field.
	TLSCrtDataName = "tls.crt"
)
