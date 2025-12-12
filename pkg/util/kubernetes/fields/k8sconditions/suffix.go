package k8sconditions

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

func appendCreatedOrUpdatedSuffix(name string) string {
	return fmt.Sprintf("%s %s", name, createdOrUpdatedSuffix)
}

func appendOutdatedSuffix(name string) string {
	return fmt.Sprintf("%s %s", name, outdatedSuffix)
}
