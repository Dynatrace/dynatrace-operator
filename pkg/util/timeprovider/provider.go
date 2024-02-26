package timeprovider

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// New time provider always returns the current time
func New() *Provider {
	return &Provider{
		now: nil,
	}
}

// The purpose if the Provider is to have a non-moving resp. fakable now for testing.
type Provider struct {
	now *metav1.Time
}

func (timeProvider *Provider) Now() *metav1.Time {
	if timeProvider.now != nil {
		return timeProvider.now
	}

	return Now()
}

func (timeProvider *Provider) Freeze() *Provider {
	timeProvider.now = Now()

	return timeProvider
}

func (timeProvider *Provider) Set(now *metav1.Time) {
	timeProvider.now = now
}

func (timeProvider *Provider) IsOutdated(previous *metav1.Time, timeout time.Duration) bool {
	return previous == nil || TimeoutReached(previous, timeProvider.Now(), timeout)
}

func TimeoutReached(previous, current *metav1.Time, timeout time.Duration) bool {
	return previous == nil || current == nil || !previous.Add(timeout).After(current.Time)
}

func Now() *metav1.Time {
	now := metav1.Now()

	return &now
}
