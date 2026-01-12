package k8sconditions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type FakeObject struct {
	metav1.Object
	Generation       int64
	StatusConditions []metav1.Condition
}

func (f *FakeObject) GetGeneration() int64 { return f.Generation }

func (f *FakeObject) Conditions() *[]metav1.Condition { return &f.StatusConditions }

func TestSetDeploymentsApplied(t *testing.T) {
	tests := []struct {
		name   string
		names  []string
		expect string
	}{
		{"non-truncated", []string{"a", "b", "c"}, "a, b, c"},
		{"truncated", []string{"a", "b", "c", "d", "e"}, "a, b, c, ... 2 more omitted"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := &FakeObject{Generation: 123}
			SetDeploymentsApplied(obj, "Test", tt.names)
			require.Len(t, obj.StatusConditions, 1)
			assert.Equal(t, tt.expect, obj.StatusConditions[0].Message)
			assert.Equal(t, int64(123), obj.StatusConditions[0].ObservedGeneration)
		})
	}
}
