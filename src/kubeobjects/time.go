package kubeobjects

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func IsOutdated(last, now *metav1.Time, threshold time.Duration) bool {
	return last == nil || last.Add(threshold).Before(now.Time)
}
