package conditions

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
			var conditions []metav1.Condition
			SetDeploymentsApplied(&conditions, "Test", tt.names)
			require.Len(t, conditions, 1)
			require.Equal(t, tt.expect, conditions[0].Message)
		})
	}
}
