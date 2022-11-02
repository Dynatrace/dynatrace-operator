package kubeobjects

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewTimeProvider() *TimeProvider {
	now := metav1.Now()
	return &TimeProvider{
		now: &now,
	}
}

type TimeProvider struct {
	now *metav1.Time
}

func (timeProvider *TimeProvider) Now() *metav1.Time {
	if timeProvider.now == nil {
		now := metav1.Now()
		timeProvider.now = &now
	}
	return timeProvider.now
}

func (timeProvider *TimeProvider) SetNow(now *metav1.Time) {
	timeProvider.now = now
}

func (timeProvider *TimeProvider) IsOutdated(previous *metav1.Time, threshold time.Duration) bool {
	return IsOutdated(previous, timeProvider.Now(), threshold)
}

func IsOutdated(previous, current *metav1.Time, threshold time.Duration) bool {
	return previous == nil || previous.Add(threshold).Before(current.Time)
}
