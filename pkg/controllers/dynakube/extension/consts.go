package extension

const (
	// secret
	eecTokenSecretKey         = "eec-token"
	eecTokenSecretValuePrefix = "EEC dt0x01"
	secretSuffix              = "-extensions-token"

	// conditions
	secretConditionType         = "ExtensionsTokenSecret"
	secretCreatedReason         = "SecretCreated"
	secretCreatedMessageSuccess = "EEC token created"
	secretCreatedMessageFailure = "Error creating extensions secret: %s"
)
