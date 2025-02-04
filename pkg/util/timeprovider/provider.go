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

	return now()
}

func (timeProvider *Provider) Freeze() *Provider {
	timeProvider.now = now()

	return timeProvider
}

func (timeProvider *Provider) Set(now time.Time) {
	_now := metav1.NewTime(now)
	timeProvider.now = &_now
}

func (timeProvider *Provider) IsOutdated(previous *metav1.Time, timeout time.Duration) bool {
	return previous == nil || TimeoutReached(previous, timeProvider.Now(), timeout)
}

func TimeoutReached(previous, current *metav1.Time, timeout time.Duration) bool {
	return previous == nil || current == nil || !previous.Add(timeout).After(current.Time)
}

func now() *metav1.Time {
	now := metav1.Now()

	return &now
}
