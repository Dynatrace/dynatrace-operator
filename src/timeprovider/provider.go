package timeprovider

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func New() *Provider {
	now := metav1.Now()
	return &Provider{
		now: &now,
	}
}

// The purpose if the Provider is to have a non-moving resp. fakable now for testing.
type Provider struct {
	now *metav1.Time
}

// Warning: This method should be never called twice on the same instance in production code where you need an accurate "now" timestamp,
// because it returns the same timestamp on each call, even if the stored "now" is hours in the past
func (timeProvider *Provider) Now() *metav1.Time {
	if timeProvider.now == nil {
		timeProvider.now = Now()
	}
	return timeProvider.now
}

func (timeProvider *Provider) SetNow(now *metav1.Time) {
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
