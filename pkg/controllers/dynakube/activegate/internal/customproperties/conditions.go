package customproperties

const customPropertiesConditionType string = "CustomProperties"

func customPropertiesConditionTypeString(customPropertiesOwnerName string) string {
	return customPropertiesConditionType + "-" + customPropertiesOwnerName
}
