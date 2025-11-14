package exp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFeatureFlagsIsOTLPInjectionSetNoProxy(t *testing.T) {
	type fields struct {
		annotations map[string]string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name:   "annotation not set",
			fields: fields{annotations: map[string]string{}},
			want:   true,
		},
		{
			name:   "annotation set to false",
			fields: fields{annotations: map[string]string{OTLPInjectionSetNoProxy: "false"}},
			want:   false,
		},
		{
			name:   "annotation set to true",
			fields: fields{annotations: map[string]string{OTLPInjectionSetNoProxy: "true"}},
			want:   true,
		},
		{
			name:   "annotation set to TRUE (case insensitive)",
			fields: fields{annotations: map[string]string{OTLPInjectionSetNoProxy: "TRUE"}},
			want:   true,
		},
		{
			name:   "annotation set to random value",
			fields: fields{annotations: map[string]string{OTLPInjectionSetNoProxy: "random"}},
			want:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ff := &FeatureFlags{
				annotations: tt.fields.annotations,
			}
			assert.Equalf(t, tt.want, ff.IsOTLPInjectionSetNoProxy(), "IsOTLPInjectionSetNoProxy()")
		})
	}
}
