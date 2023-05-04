package istio

import (
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMapErrorToObjectProbeResult(t *testing.T) {
	errorObjectNotFound := &errors.StatusError{ErrStatus: metav1.Status{Reason: metav1.StatusReasonNotFound}}
	errorTypeNotFound := &meta.NoResourceMatchError{}
	errorUnknown := fmt.Errorf("")

	tests := []struct {
		name     string
		argument error
		want     kubeobjects.ProbeResult
		wantErr  bool
	}{
		{"no error returns probeObjectFound", nil, kubeobjects.ProbeObjectFound, false},
		{"object not found error returns probeObjectNotFound", errorObjectNotFound, kubeobjects.ProbeObjectNotFound, true},
		{"type not found error returns probeTypeNotFound", errorTypeNotFound, kubeobjects.ProbeTypeNotFound, true},
		{"unknown error returns probeUnknown", errorUnknown, kubeobjects.ProbeUnknown, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := kubeobjects.MapErrorToObjectProbeResult(tt.argument)
			if (err != nil) != tt.wantErr {
				t.Errorf("mapErrorToObjectProbeResult() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("mapErrorToObjectProbeResult() got = %v, want %v", got, tt.want)
			}
		})
	}
}
