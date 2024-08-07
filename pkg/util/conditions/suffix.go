package conditions

import "fmt"

const (
	createdSuffix  = "created"
	updatedSuffix  = "updated"
	outdatedSuffix = "outdated"
)

func appendCreatedSuffix(name string) string {
	return fmt.Sprintf("%s %s", name, createdSuffix)
}

func appendUpdatedSuffix(name string) string {
	return fmt.Sprintf("%s %s", name, updatedSuffix)
}

func appendOutdatedSuffix(name string) string {
	return fmt.Sprintf("%s %s", name, outdatedSuffix)
}
