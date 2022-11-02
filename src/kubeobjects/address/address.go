package address

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

type scalarType interface {
	bool | int | int64 | time.Time | metav1.Time
}

func Of[T scalarType](i T) *T {
	return &i
}
