package consts

const (

	// TLSKeyDataName is the key used to store a TLS private key in the secret's data field.
	TLSKeyDataName = "tls.key"

	// TLSCrtDataName is the key used to store a TLS certificate in the secret's data field.
	TLSCrtDataName = "tls.crt"

	// ActiveGateCertDataName is the key used to store ActiveGate certificate data in the secret's data field.
	ActiveGateCertDataName = "activegate-tls.crt"
)
