package k8ssecuritycontext

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/version"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

const containerName = "test"

func TestGetAppArmorProfile(t *testing.T) {
	tests := []struct {
		name         string
		minorVersion int
		annotations  map[string]string
		want         *corev1.AppArmorProfile
	}{
		{
			name:         "version too old",
			minorVersion: 30,
			annotations: map[string]string{
				corev1.DeprecatedAppArmorBetaContainerAnnotationKeyPrefix + containerName: corev1.DeprecatedAppArmorBetaProfileRuntimeDefault,
			},
		},
		{
			name:         "no matching key",
			minorVersion: 31,
			annotations: map[string]string{
				"foo": "bar",
			},
		},
		{
			name:         "default",
			minorVersion: 31,
			annotations: map[string]string{
				corev1.DeprecatedAppArmorBetaContainerAnnotationKeyPrefix + containerName: corev1.DeprecatedAppArmorBetaProfileRuntimeDefault,
			},
			want: &corev1.AppArmorProfile{
				Type: corev1.AppArmorProfileTypeRuntimeDefault,
			},
		},
		{
			name:         "unconfined",
			minorVersion: 31,
			annotations: map[string]string{
				corev1.DeprecatedAppArmorBetaContainerAnnotationKeyPrefix + containerName: corev1.DeprecatedAppArmorBetaProfileNameUnconfined,
			},
			want: &corev1.AppArmorProfile{
				Type: corev1.AppArmorProfileTypeUnconfined,
			},
		},
		{
			name:         "localhost",
			minorVersion: 31,
			annotations: map[string]string{
				corev1.DeprecatedAppArmorBetaContainerAnnotationKeyPrefix + containerName: corev1.DeprecatedAppArmorBetaProfileNamePrefix + "foo",
			},
			want: &corev1.AppArmorProfile{
				Type:             corev1.AppArmorProfileTypeLocalhost,
				LocalhostProfile: ptr.To("foo"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(version.DisableCacheForTest(tt.minorVersion))
			got := GetAppArmorProfile(tt.annotations, containerName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRemoveAppArmorAnnotations(t *testing.T) {
	tests := []struct {
		name         string
		minorVersion int
		annotations  map[string]string
		want         map[string]string
	}{
		{
			name:         "version too old",
			minorVersion: 30,
			annotations: map[string]string{
				corev1.DeprecatedAppArmorBetaContainerAnnotationKeyPrefix + containerName: corev1.DeprecatedAppArmorBetaProfileRuntimeDefault,
			},
			want: map[string]string{
				corev1.DeprecatedAppArmorBetaContainerAnnotationKeyPrefix + containerName: corev1.DeprecatedAppArmorBetaProfileRuntimeDefault,
			},
		},
		{
			name:         "no matching keys",
			minorVersion: 31,
			annotations: map[string]string{
				"foo": "bar",
			},
			want: map[string]string{
				"foo": "bar",
			},
		},
		{
			name:         "matching keys",
			minorVersion: 31,
			annotations: map[string]string{
				corev1.DeprecatedAppArmorBetaContainerAnnotationKeyPrefix + "foo": corev1.DeprecatedAppArmorBetaProfileRuntimeDefault,
				corev1.DeprecatedAppArmorBetaContainerAnnotationKeyPrefix + "bar": corev1.DeprecatedAppArmorBetaProfileRuntimeDefault,
				"foo": corev1.DeprecatedAppArmorBetaProfileRuntimeDefault,
			},
			want: map[string]string{
				"foo": corev1.DeprecatedAppArmorBetaProfileRuntimeDefault,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(version.DisableCacheForTest(tt.minorVersion))
			got := RemoveAppArmorAnnotations(tt.annotations)
			assert.Equal(t, tt.want, got)
		})
	}
}
