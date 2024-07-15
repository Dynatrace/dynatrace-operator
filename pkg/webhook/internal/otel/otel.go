package otel

import (
	"os"
)

var WebhookPodName string

func init() {
	WebhookPodName = os.Getenv("POD_NAME")
}
