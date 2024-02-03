package otel

import (
	"os"
	"sync"
)

var envPodName string
var oncePodName = sync.Once{}

func GetWebhookPodName() string {
	oncePodName.Do(func() {
		envPodName = os.Getenv("POD_NAME")
	})

	return envPodName
}
