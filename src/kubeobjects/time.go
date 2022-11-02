package kubeobjects

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func IsOutdated(previous, current *metav1.Time, threshold time.Duration) bool {
	return previous == nil || previous.Add(threshold).Before(current.Time)
}
