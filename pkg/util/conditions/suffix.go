package conditions

import "fmt"

const (
	createdSuffix          = "created"
	updatedSuffix          = "updated"
	createdOrUpdatedSuffix = "created/updated"
	outdatedSuffix         = "outdated"
)

func appendCreatedSuffix(name string) string {
	return fmt.Sprintf("%s %s", name, createdSuffix)
}

func appendUpdatedSuffix(name string) string {
	return fmt.Sprintf("%s %s", name, updatedSuffix)
}

func appendCreatedOrUpdatedSuffix(name string) string {
	return fmt.Sprintf("%s %s", name, createdOrUpdatedSuffix)
}

func appendOutdatedSuffix(name string) string {
	return fmt.Sprintf("%s %s", name, outdatedSuffix)
}
