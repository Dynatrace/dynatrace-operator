package conditions

import "fmt"

const (
	createdSuffix  = "created"
	updatedSuffix  = "updated"
	deletedSuffix  = "deleted"
	outdatedSuffix = "outdated"
)

func appendCreatedSuffix(conditionType string) string {
	return fmt.Sprintf("%s %s", conditionType, createdSuffix)
}

func appendUpdatedSuffix(conditionType string) string {
	return fmt.Sprintf("%s %s", conditionType, updatedSuffix)
}

func appendDeletedSuffix(conditionType string) string {
	return fmt.Sprintf("%s %s", conditionType, deletedSuffix)
}

func appendOutdatedSuffix(conditionType string) string {
	return fmt.Sprintf("%s %s", conditionType, outdatedSuffix)
}
