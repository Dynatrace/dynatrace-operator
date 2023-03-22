package address

import (
	"time"

	"golang.org/x/exp/constraints"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type scalarType interface {
	bool | constraints.Integer | constraints.Float | time.Time | metav1.Time
}

func Of[T scalarType](i T) *T {
	return &i
}
