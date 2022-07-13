package kubeobjects

import "os"

func IsRunLocally() bool {
	return os.Getenv("RUN_LOCAL") == "true"
}
